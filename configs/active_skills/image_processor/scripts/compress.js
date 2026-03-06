// 图片压缩
const sharp = require('sharp');
const fs = require('fs');

const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun compress.js <input> <output> [quality]");
  console.log("示例: bun compress.js photo.jpg compressed.jpg 80");
  process.exit(1);
}

const [input, output, qualityStr = '80'] = args;
const quality = parseInt(qualityStr);

if (isNaN(quality) || quality < 1 || quality > 100) {
  console.error("❌ 质量必须在 1-100 之间");
  process.exit(1);
}

async function compressImage() {
  try {
    console.log(`压缩图片: ${input} (质量: ${quality}%)`);
    
    const inputStats = fs.statSync(input);
    console.log(`   原始大小: ${(inputStats.size / 1024).toFixed(2)} KB`);
    
    await sharp(input)
      .jpeg({ quality, mozjpeg: true }) // 使用 mozjpeg 更好的压缩
      .toFile(output);
    
    const outputStats = fs.statSync(output);
    const compressionRatio = ((1 - outputStats.size / inputStats.size) * 100).toFixed(1);
    
    console.log(`✅ 压缩完成: ${output}`);
    console.log(`   压缩后大小: ${(outputStats.size / 1024).toFixed(2)} KB`);
    console.log(`   压缩率: ${compressionRatio}%`);
    
  } catch (error) {
    console.error("❌ 压缩失败:", error.message);
    process.exit(1);
  }
}

compressImage();
