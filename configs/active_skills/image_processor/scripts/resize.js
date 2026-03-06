// 图片调整大小
const sharp = require('sharp');
const fs = require('fs');

const args = process.argv.slice(2);
if (args.length < 4) {
  console.log("用法: bun resize.js <input> <width> <height> <output>");
  console.log("示例: bun resize.js photo.jpg 800 600 resized.jpg");
  process.exit(1);
}

const [input, widthStr, heightStr, output] = args;
const width = parseInt(widthStr);
const height = parseInt(heightStr);

if (isNaN(width) || isNaN(height)) {
  console.error("❌ 宽度和高度必须是数字");
  process.exit(1);
}

async function resizeImage() {
  try {
    console.log(`调整图片大小: ${input} -> ${width}x${height}`);
    
    await sharp(input)
      .resize(width, height, {
        fit: 'inside', // 保持宽高比
        withoutEnlargement: true // 不放大小图
      })
      .toFile(output);
    
    const stats = fs.statSync(output);
    console.log(`✅ 调整完成: ${output}`);
    console.log(`   文件大小: ${(stats.size / 1024).toFixed(2)} KB`);
    
  } catch (error) {
    console.error("❌ 处理失败:", error.message);
    process.exit(1);
  }
}

resizeImage();
