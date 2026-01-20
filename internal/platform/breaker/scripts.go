package breaker

import "hauslet/internal/platform/redis"

var (
	allowRequestScript = redis.NewScript(`
local state = redis.call("GET", KEYS[1])
if not state or state == "closed" then
    return 1
end

if state == "open" then
    local openedAt = tonumber(redis.call("GET", KEYS[2]))
    if not openedAt then return 0 end

    local now = tonumber(ARGV[1])
    local timeout = tonumber(ARGV[2])
    
    if (now - openedAt) >= timeout then
        redis.call("SET", KEYS[1], "half_open")
        return 1
    end
    return 0
end

if state == "half_open" then
    local count = redis.call("INCR", KEYS[3])
    redis.call("EXPIRE", KEYS[3], tonumber(ARGV[4]))
    
    local max = tonumber(ARGV[3])
    if count <= max then
        return 1
    end
    return 0
end

return 1
`)

	recordSuccessScript = redis.NewScript(`
local state = redis.call("GET", KEYS[1])
local successCount = redis.call("INCR", KEYS[2])
redis.call("SET", KEYS[3], ARGV[2])

if state == "half_open" then
    local threshold = tonumber(ARGV[1])
    if successCount >= threshold then
        redis.call("SET", KEYS[1], "closed")
        redis.call("DEL", KEYS[2], KEYS[4], KEYS[5], KEYS[6])
        return 1
    end
end
return 0
`)

	recordFailureScript = redis.NewScript(`
local state = redis.call("GET", KEYS[1])
local failCount = redis.call("INCR", KEYS[2])
redis.call("SET", KEYS[3], ARGV[2])

if state == "half_open" then
    redis.call("SET", KEYS[1], "open")
    redis.call("SET", KEYS[4], ARGV[2])
    redis.call("DEL", KEYS[5], KEYS[6])
    return 1
end

if not state or state == "closed" then
    local threshold = tonumber(ARGV[1])
    if failCount >= threshold then
        redis.call("SET", KEYS[1], "open")
        redis.call("SET", KEYS[4], ARGV[2])
        return 1
    end
end

return 0
`)
)
