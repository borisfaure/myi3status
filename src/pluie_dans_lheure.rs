use crate::I3ProtocolBlock;
use anyhow::{anyhow, Result};
use reqwest::header::{HeaderMap, HeaderValue, AUTHORIZATION, SET_COOKIE, USER_AGENT};
use serde_json::Value;
use std::time::{SystemTime, UNIX_EPOCH};
use tokio::fs::File;
use tokio::io::{AsyncReadExt, AsyncWriteExt};

const STATUS_LEN: usize = 9;
const USER_AGENT_VALUE: &str = "PluieDansLheureFori3status";
const FILE_PATH: &str = "/tmp/pluie_dans_lheure";

async fn get_bearer() -> Result<String> {
    let url = "https://meteofrance.com/previsions-meteo-france/";
    let client = reqwest::Client::new();
    let res = client
        .head(url)
        .header(USER_AGENT, USER_AGENT_VALUE)
        .send()
        .await?;

    let set_cookie = res
        .headers()
        .get(SET_COOKIE)
        .ok_or_else(|| anyhow!("No Set-Cookie header"))?
        .to_str()?;

    if !set_cookie.starts_with("mfsession=") {
        return Err(anyhow!("No MFSESSION in Set-Cookie"));
    }
    let cookie = set_cookie.trim_start_matches("mfsession=");
    let cookie = cookie
        .split(';')
        .next()
        .ok_or_else(|| anyhow!("No cookie"))?;

    let rot13_encoded = cookie
        .chars()
        .map(|c| {
            if c.is_ascii_alphabetic() {
                let offset = if c.is_ascii_lowercase() { b'a' } else { b'A' };
                let rotated = (c as u8 - offset + 13) % 26 + offset;
                rotated as char
            } else {
                c
            }
        })
        .collect::<String>();

    Ok(format!("Bearer {}", rot13_encoded))
}

async fn get_status_from_http(location: String) -> Result<String> {
    let url = format!(
        "https://rpcache-aa.meteofrance.com/internet2018client/2.0/nowcast/rain?{}",
        location
    );
    let bearer = get_bearer().await?;
    let client = reqwest::Client::new();

    let mut headers = HeaderMap::new();
    headers.insert(USER_AGENT, HeaderValue::from_static(USER_AGENT_VALUE));
    headers.insert(AUTHORIZATION, HeaderValue::from_str(&bearer)?);

    let res = client.get(&url).headers(headers).send().await?;
    let body = res.text().await?;
    let data: Value = serde_json::from_str(&body)?;

    if !data.is_object() {
        return Err(anyhow!("Not an object"));
    }
    let properties = data["properties"]
        .as_object()
        .ok_or_else(|| anyhow!("No properties"))?;
    let forecast = properties["forecast"]
        .as_array()
        .ok_or_else(|| anyhow!("No forecast"))?;
    let mut rain = String::with_capacity(forecast.len());
    for f in forecast {
        let f = f.as_object().ok_or_else(|| anyhow!("No forecast object"))?;
        let intensity = f["rain_intensity"]
            .as_f64()
            .ok_or_else(|| anyhow!("No intensity"))?;
        let rune = if intensity <= 1. {
            '_'
        } else if intensity <= 2. {
            '░'
        } else if intensity <= 3. {
            '▒'
        } else if intensity <= 4. {
            '▓'
        } else {
            '█'
        };
        rain.push(rune);
    }
    Ok(rain)
}

async fn need_new_status(file: &mut File, location: String) -> Result<String> {
    let status = get_status_from_http(location).await?;
    file.set_len(0).await?;
    file.write_all(status.as_bytes()).await?;
    file.sync_all().await?;
    Ok(status)
}

async fn get_rain_string(location: String) -> Result<String> {
    // Open file for reading and writing, creating it if it doesn't exist
    // If the file is too small or too old, fetch a new status
    let mut file = File::options()
        .write(true)
        .read(true)
        .create(true)
        .truncate(false)
        .open(FILE_PATH)
        .await?;

    let metadata = file.metadata().await?;
    let modified = metadata.modified()?;
    let now = SystemTime::now().duration_since(UNIX_EPOCH)?;

    if metadata.len() < STATUS_LEN as u64
        || now.as_secs() - modified.duration_since(UNIX_EPOCH)?.as_secs() > 300
    {
        need_new_status(&mut file, location).await
    } else {
        let mut status: Vec<u8> = Vec::with_capacity(STATUS_LEN);
        file.read_to_end(&mut status).await?;
        Ok(status.iter().map(|&c| c as char).collect::<String>())
    }
}

pub async fn get_rain_i3bar_format(
    location: String,
    rain_color: String,
) -> Option<I3ProtocolBlock> {
    let status = get_rain_string(location).await.ok()?;

    Some(I3ProtocolBlock {
        name: "pluie_dans_lheure".to_string(),
        full_text: status,
        color: Some(rain_color),
        ..Default::default()
    })
}
