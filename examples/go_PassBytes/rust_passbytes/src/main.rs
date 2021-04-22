use image::GenericImageView;
use ssvm_wasi_helper::ssvm_wasi_helper::_initialize;
use ssvm_wasi_helper::ssvm_wasi_helper::get_bytes_from_caller;

fn main() {
    println!("Rust: Process start.");

    _initialize();
    let data = match get_bytes_from_caller() {
        Ok(buf) => buf,
        Err(e) => {
            println!("Rust: Get bytes failed: {}", e);
            println!("Rust: Process end.");
            return;
        }
    };

    
    match image::load_from_memory(&data) {
        Ok(img) => {
            println!("Rust: Got image data from stdin. Width: {}, Height: {}", img.width(), img.height());
        }
        Err(e) => println!("Rust: ERROR - Parse image failed: {}", e),
    }
    println!("Rust: Process end.");
}
