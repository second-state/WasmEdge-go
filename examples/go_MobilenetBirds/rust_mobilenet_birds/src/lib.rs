use wasm_bindgen::prelude::*;
use wasmedge_tensorflow_interface;
use std::time::{Instant};

#[wasm_bindgen]
pub fn infer(image_data: &[u8]) -> String {
    let start = Instant::now();
    let img = image::load_from_memory(image_data).unwrap().to_rgb8();
    println!("RUST: Loaded image in ... {:?}", start.elapsed());
    let resized = image::imageops::thumbnail(&img, 192, 192);
    println!("RUST: Resized image in ... {:?}", start.elapsed());
    // let resized = image::imageops::resize(&img, 224, 224, ::image::imageops::FilterType::Triangle);
    // let resized = image::imageops::resize(&img, 224, 224, ::image::imageops::FilterType::Nearest);
    let mut flat_img: Vec<f32> = Vec::new();
    for rgb in resized.pixels() {
        flat_img.push(rgb[0] as f32 / 255.);
        flat_img.push(rgb[1] as f32 / 255.);
        flat_img.push(rgb[2] as f32 / 255.);
    }

    let model_data: &[u8] = include_bytes!("mobilenet_v2_192res_1.0_inat_bird.pb");
    let labels = include_str!("aiy_birds_V1_labelmap.txt");

    let mut session = wasmedge_tensorflow_interface::Session::new(model_data, wasmedge_tensorflow_interface::ModelType::TensorFlow);
    session.add_input("map/TensorArrayStack/TensorArrayGatherV3", &flat_img, &[1, 192, 192, 3])
           .add_output("prediction")
           .run();
    let res_vec: Vec<f32> = session.get_output("prediction");
    println!("RUST: Parsed output in ... {:?}", start.elapsed());

    let mut i = 0;
    let mut max_index: i32 = -1;
    let mut max_value: f32 = -1.0;
    while i < res_vec.len() {
        let cur = res_vec[i];
        if cur > max_value {
            max_value = cur;
            max_index = i as i32;
        }
        i += 1;
    }
    println!("RUST: index {}, prob {}", max_index, max_value);

    let confidence: String;
    if max_value > 0.75 {
        confidence = "is very likely".to_string();
    } else if max_value > 0.5 {
        confidence = "is likely".to_string();
    } else {
        confidence = "could be".to_string();
    }

    let ret_str: String;
    if max_value > 0.2 {
        let mut label_lines = labels.lines();
        for _i in 0..max_index {
          label_lines.next();
        }
        let bird_name = label_lines.next().unwrap().to_string();
        ret_str = format!("It {} a <a href='https://www.google.com/search?q={}'>{}</a> in the picture", confidence, bird_name, bird_name);
    } else {
        ret_str = "It does not appears to be a bird in the picture.".to_string();
    }
    
    println!("RUST: Finished post-processing in ... {:?}", start.elapsed());
    return ret_str;
}
