-- 热榜读写混合压测脚本
-- 60% 热榜查询
-- 20% 文章详情（触发浏览量 +1）
-- 10% 点赞
-- 5% 收藏
-- 5% 评论

local counter = 0

function init(args)
    math.randomseed(os.time())
end

function request()
    counter = counter + 1
    local rand = math.random(100)
    local id = math.random(1, 200000)  -- 随机文章 ID
    
    if rand <= 60 then
        -- 60% 热榜查询
        return wrk.format("GET", "/api/v1/articles?sort=hot&page=1&page_size=20")
    elseif rand <= 80 then
        -- 20% 文章详情（触发浏览量 +1）
        return wrk.format("GET", "/api/v1/articles/" .. id)
    elseif rand <= 90 then
        -- 10% 点赞
        return wrk.format("POST", "/api/v1/articles/" .. id .. "/like")
    elseif rand <= 95 then
        -- 5% 收藏
        return wrk.format("POST", "/api/v1/articles/" .. id .. "/favorite")
    else
        -- 5% 评论
        local headers = {
            ["Content-Type"] = "application/json"
        }
        local body = '{"content": "测试评论 ' .. counter .. '"}'
        return wrk.format("POST", "/api/v1/articles/" .. id .. "/comments", headers, body)
    end
end

function response(status, headers, body)
    -- 可选：统计不同状态码的数量
end
