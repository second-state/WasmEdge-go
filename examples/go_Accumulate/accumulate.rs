use wasm_bindgen::prelude::*;

static mut X: i32 = 0;

#[wasm_bindgen]
pub fn acc() {
    unsafe {
        X = X + 1;
        println!("{}", X);
    }
}
