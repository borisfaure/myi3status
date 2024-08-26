use crate::I3ProtocolBlock;
use tokio::process::Command;

pub async fn get_current_playing() -> Option<I3ProtocolBlock> {
    let child_res = Command::new("spotifyctl").arg("get").output().await;
    if child_res.is_err() {
        return None;
    }
    let song = child_res.unwrap().stdout;
    let song = String::from_utf8(song).unwrap_or("".to_string());
    if song.is_empty() {
        None
    } else {
        Some(I3ProtocolBlock {
            name: "spotify".to_string(),
            full_text: song.trim().to_string(),
            ..Default::default()
        })
    }
}
