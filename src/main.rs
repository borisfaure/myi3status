use clap::{Arg, Command as ClapCommand};
use serde::{Deserialize, Serialize};
use std::error::Error;
use tokio::{
    io::{AsyncBufReadExt, BufReader},
    process::Command,
};

mod spotify;

#[derive(Debug, Serialize, Deserialize)]
struct I3ProtocolHeader {
    version: i32,
}

#[derive(Default, Debug, Serialize, Deserialize)]
pub struct I3ProtocolBlock {
    pub name: String,
    pub full_text: String,
    #[serde(skip_serializing_if = "Option::is_none")]
    instance: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    color: Option<String>,
    #[serde(skip_serializing_if = "Option::is_none")]
    markup: Option<String>,
}

/// Command line options
fn build_cli() -> ClapCommand {
    ClapCommand::new("i3status_rust")
        .arg(
            Arg::new("location")
                .long("location")
                .help("Location for the Pluie dans l'heure API")
                .num_args(1)
                .required(true),
        )
        .arg(
            Arg::new("rain_color")
                .long("rain-color")
                .help("Color to display text when it's raining")
                .num_args(1)
                .default_value("#268bd2"),
        )
}

async fn get_rain_i3bar_format(_location: String, _rain_color: String) -> Option<I3ProtocolBlock> {
    // Implement this function to get weather data
    // Return Some(I3ProtocolBlock) or None
    None // Placeholder
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn Error>> {
    let matches = build_cli().get_matches();
    let location = matches.get_one::<String>("location").unwrap();
    let rain_color = matches.get_one::<String>("rain_color").unwrap();

    let mut cmd = Command::new("i3status")
        .kill_on_drop(true)
        .stdout(std::process::Stdio::piped())
        .stderr(std::process::Stdio::piped())
        .spawn()?;
    let stdout = cmd.stdout.take().expect("Failed to get stdout");
    let mut reader = BufReader::new(stdout).lines();

    // Read and parse header
    let line = reader.next_line().await?.unwrap();
    let header: I3ProtocolHeader = serde_json::from_str(&line)?;
    if header.version != 1 {
        return Err("Invalid header version".into());
    }
    println!("{}", serde_json::to_string(&header)?);

    // Check for opening bracket
    let line = reader.next_line().await?.unwrap();
    if line != "[" {
        return Err("Invalid second line".into());
    }
    println!("[");

    let mut first = true;
    loop {
        let mut line = reader.next_line().await?.unwrap();
        if !line.is_empty() && line.starts_with(",") {
            // skip comma
            line.remove(0);
        }
        let mut blocks: Vec<I3ProtocolBlock> = serde_json::from_str(&line)?;

        let rain = get_rain_i3bar_format(location.clone(), rain_color.clone()).await;
        let song = spotify::get_current_playing().await;

        if let Some(rain) = rain {
            blocks.insert(0, rain);
        }
        if let Some(song) = song {
            blocks.insert(0, song);
        }

        let json = serde_json::to_string(&blocks)?;
        if first {
            println!("{}", json);
            first = false;
        } else {
            println!(",{}", json);
        }
    }
}
