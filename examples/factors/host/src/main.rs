use bonsai_sdk::alpha as bonsai_sdk;
use methods::FACTORS_ELF;
use risc0_zkvm::{MemoryImage, Program, GUEST_MAX_MEM, PAGE_SIZE};

fn main() -> anyhow::Result<()> {
    // Initialize tracing. In order to view logs, run `RUST_LOG=info cargo run`
    env_logger::init();

    let client = bonsai_sdk::Client::from_env(risc0_zkvm::VERSION)?;

    // Create the memoryImg, upload it and return the imageId
    let img_id = {
        // TODO revisit with 0.20. Seems like the memory image generation should be streamlined.
        let program = Program::load_elf(FACTORS_ELF, GUEST_MAX_MEM as u32)?;
        let image = MemoryImage::new(&program, PAGE_SIZE as u32)?;
        // TODO check if the image id has to be this exact form. Seems strange.
        let image_id = hex::encode(image.compute_id());
        let image = bincode::serialize(&image).expect("Failed to serialize memory img");
        client.upload_img(&image_id, image)?;
        image_id
    };

    println!("Uploaded image with id: {}", img_id);

    // TODO move this to a separate binary. Currently just used to quickly check to make sure Receipt is good.
    // let receipt = include_bytes!("../../../../receipt.bin");
    // let receipt = bincode::deserialize::<risc0_zkvm::Receipt>(receipt)?;
    // let result: u64 = receipt.journal.decode()?;
    // println!("Result: {}", result);
    // receipt.verify(methods::FACTORS_ID)?;

    Ok(())
}
