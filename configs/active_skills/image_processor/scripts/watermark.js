// 添加水印
const sharp = require('sharp');

const args = process.argv.slice(2);
if (args.length < 3) {
  console.log("用法: bun watermark.js <input> <watermark> <output>");
  console.log("示例: bun watermark.js photo.jpg logo.png watermarked.jpg");
  process.exit(1);
}

const [input, watermarkPath, output] = args;

async function addWatermark() {
  try {
    console.log(`添加水印: ${watermarkPath} -> ${input}`);
    
    // 读取主图片和水印
    const mainImage = sharp(input);
    const { width, height } = await mainImage.metadata();
    
    // 调整水印大小（主图的 20%）
    const watermarkSize = Math.min(width, height) * 0.2;
    const watermark = await sharp(watermarkPath)
      .resize(watermarkSize, watermarkSize)
      .composite([{
        input: Buffer.from([255, 255, 255, 128]), // 半透明白色背景
        blend: 'dest-in'
      }])
      .toBuffer();
    
    // 合成图片（右下角）
    await mainImage
      .composite([{
        input: watermark,
        gravity: 'southeast', // 右下角
        top: height - watermarkSize - 20,
        left: width - watermarkSize - 20
      }])
      .toFile(output);
    
    console.log(`✅ 水印添加完成: ${output}`);
    
  } catch (error) {
    console.error("❌ 添加水印失败:", error.message);
    process.exit(1);
  }
}

addWatermark();
