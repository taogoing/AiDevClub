-- AIDevClub 分类初始化脚本
-- 用于服务器上已有数据库的分类更新

-- 备份现有文章的 category_id（如果有）
-- ALTER TABLE articles ADD COLUMN old_category_id INT;

-- 删除现有分类（注意：如果有文章关联，需要先处理）
-- 先将文章的 category_id 设为新分类的 ID
UPDATE articles SET category_id = 1 WHERE category_id IN (1, 2, 3, 4, 5, 6, 7, 8, 9);
-- 如果有其他依赖，根据实际情况调整

-- 清空分类表
TRUNCATE TABLE categories;

-- 插入新分类
INSERT INTO categories (name, slug, sort_order, created_at, updated_at) VALUES
('Go', 'go', 1, NOW(), NOW()),
('后端', 'backend', 2, NOW(), NOW()),
('前端', 'frontend', 3, NOW(), NOW()),
('AI', 'ai', 4, NOW(), NOW()),
('Agent', 'agent', 5, NOW(), NOW()),
('数据库', 'database', 6, NOW(), NOW()),
('DevOps', 'devops', 7, NOW(), NOW());

-- 验证
SELECT * FROM categories ORDER BY sort_order;
