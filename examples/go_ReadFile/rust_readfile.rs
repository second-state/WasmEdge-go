use std::env;
use std::fs::File;
use std::io::{self, BufRead};

fn main() {
    // Get the argv.
    let args: Vec<String> = env::args().collect();
    if args.len() <= 1 {
        println!("Rust: ERROR - No input file name.");
        return;
    }

    // Open the file.
    println!("Rust: Opening input file \"{}\"...", args[1]);
    let file = match File::open(&args[1]) {
        Err(why) => {
            println!("Rust: ERROR - Open file \"{}\" failed: {}", args[1], why);
            return;
        },
        Ok(file) => file,
    };

    // Read lines.
    let reader = io::BufReader::new(file);
    let mut texts:Vec<String> = Vec::new();
    for line in reader.lines() {
        if let Ok(text) = line {
            texts.push(text);
        }
    }
    println!("Rust: Read input file \"{}\" succeeded.", args[1]);

    // Get stdin to print lines.
    println!("Rust: Please input the line number to print the line of file.");
    let stdin = io::stdin();
    for line in stdin.lock().lines() {
        let input = line.unwrap();
        match input.parse::<usize>() {
            Ok(n) => if n > 0 && n <= texts.len() {
                println!("{}", texts[n - 1]);
            } else {
                println!("Rust: ERROR - Line \"{}\" is out of range.", n);
            },
            Err(e) => println!("Rust: ERROR - Input \"{}\" is not an integer: {}", input, e),
        }
    }
    println!("Rust: Process end.");
}
