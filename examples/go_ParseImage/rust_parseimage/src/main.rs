use image::GenericImageView;
use std::io::{self, Read};

fn main() {
    let mut buffer: Vec<u8> = Vec::new();

    match io::stdin().read_to_end(&mut buffer) {
        Ok(n) => {
            println!("Rust: Read {} bytes from stdin.", n);
        }
        Err(e) => println!("Rust: ERROR - Read stdin failed: {}", e),
    }

    match image::load_from_memory(&buffer) {
        Ok(img) => {
            println!("Rust: Got image data from stdin. Width: {}, Height: {}", img.width(), img.height());
        }
        Err(e) => println!("Rust: ERROR - Parse image failed: {}", e),
    }
}