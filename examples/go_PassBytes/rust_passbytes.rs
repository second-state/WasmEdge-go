use std::env;
use std::fs;

fn main() {
    println!("Rust: Process start.");

    // Get the envp for the temp file path.
    let path = match env::var("TEMP_INPUT") {
        Err(_) => {
            println!("Rust: ERROR - Temp file not specified.");
            println!("Rust: Process end.");
            return;
        }, 
        Ok(val) => val,
    };

    // Read the temp file.
    let buf = match fs::read(&path) {
        Err(why) => {
            println!("Rust: ERROR - Open temp file \"{}\" failed: {}", path, why);
            println!("Rust: Process end.");
            return;
        },
        Ok(buf) => buf,
    };

    println!("Rust: Got bytes {:02X?}", buf);

    // Get the argv.
    let args: Vec<String> = env::args().collect();
    for i in 1..args.len() {
        match args[i].parse::<usize>() {
            Ok(n) => if n < buf.len() {
                println!("Rust: The index {} of bytes is {:02X?}", n, buf[n]);
            } else {
                println!("Rust: ERROR - Index \"{}\" is out of range.", n);
            },
            Err(e) => println!("Rust: ERROR - Param \"{}\" is not an integer: {}", args[i], e),
        }
    }

    println!("Rust: Process end.");
}
