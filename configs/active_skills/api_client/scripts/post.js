// POST 请求
const args = process.argv.slice(2);
if (args.length < 2) {
  console.log("用法: bun post.js <url> <json_data>");
  console.log("示例: bun post.js https://api.example.com/data '{\"name\":\"test\"}'");
  process.exit(1);
}

const [url, jsonStr] = args;

async function postRequest() {
  try {
    console.log(`POST 请求: ${url}`);
    
    let data;
    try {
      data = JSON.parse(jsonStr);
    } catch {
      console.error("❌ JSON 数据格式错误");
      process.exit(1);
    }
    
    const controller = new AbortController();
    const timeout = setTimeout(() => controller.abort(), 10000);
    
    const response = await fetch(url, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'Accept': 'application/json'
      },
      body: JSON.stringify(data),
      signal: controller.signal
    });
    
    clearTimeout(timeout);
    
    console.log(`状态码: ${response.status} ${response.statusText}`);
    
    const result = await response.json();
    console.log('\n响应数据:');
    console.log(JSON.stringify(result, null, 2));
    
    console.log('\n✅ POST 请求成功');
    
  } catch (error) {
    if (error.name === 'AbortError') {
      console.error("❌ 请求超时");
    } else {
      console.error("❌ 请求失败:", error.message);
    }
    process.exit(1);
  }
}

postRequest();
