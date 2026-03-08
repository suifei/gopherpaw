#!/bin/bash

echo "=== 测试目录映射 ==="
echo ""

# 1. 测试配置文件
echo "1. 测试配置文件..."
docker exec gopherpaw-desktop test -f /app/config.yaml && echo "✅ config.yaml 存在" || echo "❌ config.yaml 不存在"

# 2. 测试技能目录
echo ""
echo "2. 测试技能目录..."
SKILL_COUNT=$(docker exec gopherpaw-desktop ls /app/active_skills 2>/dev/null | wc -l)
if [ "$SKILL_COUNT" -ge 20 ]; then
    echo "✅ 技能目录正常 ($SKILL_COUNT 个)"
    docker exec gopherpaw-desktop ls /app/active_skills | head -5
    echo "... (共 $SKILL_COUNT 个技能)"
else
    echo "❌ 技能目录异常 ($SKILL_COUNT 个)"
fi

# 3. 测试提示词文件
echo ""
echo "3. 测试提示词文件..."
docker exec gopherpaw-desktop test -d /app/md_files/zh && echo "✅ 中文提示词存在" || echo "❌ 中文提示词不存在"
docker exec gopherpaw-desktop test -d /app/md_files/en && echo "✅ 英文提示词存在" || echo "❌ 英文提示词不存在"

# 4. 测试写入权限
echo ""
echo "4. 测试写入权限..."
docker exec gopherpaw-desktop touch /app/data/test_write.txt && echo "✅ 数据目录可写" || echo "❌ 数据目录不可写"

# 5. 测试宿主机访问
echo ""
echo "5. 测试宿主机访问..."
echo "test" > /mnt/d/works/gateway/gopherpaw/docker/data/host_test.txt
docker exec gopherpaw-desktop test -f /app/data/host_test.txt && echo "✅ 宿主机文件可见" || echo "❌ 宿主机文件不可见"
rm -f /mnt/d/works/gateway/gopherpaw/docker/data/host_test.txt

# 6. 测试配置目录
echo ""
echo "6. 测试配置目录..."
docker exec gopherpaw-desktop test -f /app/configs/config.yaml.example && echo "✅ 配置示例文件存在" || echo "❌ 配置示例文件不存在"

# 7. 测试技能文件
echo ""
echo "7. 测试技能文件（抽样）..."
docker exec gopherpaw-desktop test -f /app/active_skills/pdf/SKILL.md && echo "✅ PDF 技能文件存在" || echo "❌ PDF 技能文件不存在"
docker exec gopherpaw-desktop test -f /app/active_skills/browser_visible/SKILL.md && echo "✅ 浏览器技能文件存在" || echo "❌ 浏览器技能文件不存在"

echo ""
echo "=== 测试完成 ==="
