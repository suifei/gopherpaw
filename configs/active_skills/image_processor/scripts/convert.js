// 图片格式转换
const sharp = require('sharp');
const path = require('path');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun convert.js <input> <output>");
  console.log("示例: bun convert.js photo.png photo.webp");
  process.exit(1);
}

const [input, output] = args;

async function convertImage() {
  try {
    const inputExt = path.extname(input).toLowerCase();
    const outputExt = path.extname(output).toLowerCase();
    
    console.log(`转换图片格式: ${inputExt} -> ${outputExt}`);
    
    let image = sharp(input);
    
    // 根据输出格式设置选项
    switch (outputExt) {
      case '.jpg':
      case '.jpeg':
        image = image.jpeg({ quality: 90 });
        break;
      case '.png':
        image = image.png({ compressionLevel: 9 });
        break;
      case '.webp':
        image = image.webp({ quality: 85 });
        break;
      case '.avif':
        image = image.avif({ quality: 80 });
        break;
      default:
        console.error(`❌ 不支持的输出格式: ${outputExt}`);
        process.exit(1);
    }
    
    await image.toFile(output);
    
    const fs = require('fs');
    const inputStats = fs.statSync(input);
    const outputStats = fs.statSync(output);
    
    console.log(`✅ 转换完成: ${output}`);
    console.log(`   输入大小: ${(inputStats.size / 1024).toFixed(2)} KB`);
    console.log(`   输出大小: ${(outputStats.size / 1024).toFixed(2)} KB`);
    console.log(`   压缩率: ${((1 - outputStats.size / inputStats.size) * 100).toFixed(1)}%`);
    
  } catch (error) {
    console.error("❌ 转换失败:", error.message);
    process.exit(1);
  }
}

convertImage();
