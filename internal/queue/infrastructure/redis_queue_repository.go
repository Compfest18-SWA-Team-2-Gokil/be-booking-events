package infrastructure

import (
	"context"
	"fmt"
	"time"

	"github.com/ebk-tech/be-booking-events/internal/queue/application"
	"github.com/redis/go-redis/v9"
)

const activeQueuesKey = "active_queue_events"

type RedisQueueRepository struct {
	client *redis.Client
}

func NewRedisQueueRepository(client *redis.Client) *RedisQueueRepository {
	return &RedisQueueRepository{client: client}
}

var _ application.QueueRepository = (*RedisQueueRepository)(nil)

// Join mendaftarkan user ke antrean. NX: tidak overwrite jika sudah ada.
func (r *RedisQueueRepository) Join(ctx context.Context, eventID, userID string) (int64, error) {
	key := queueKey(eventID)
	score := float64(time.Now().UnixMilli())

	// NX: hanya tambah jika member belum ada.
	r.client.ZAddNX(ctx, key, redis.Z{Score: score, Member: userID})

	rank, err := r.client.ZRank(ctx, key, userID).Result()
	if err != nil {
		return 0, fmt.Errorf("get rank: %w", err)
	}
	return rank, nil
}

func (r *RedisQueueRepository) Position(ctx context.Context, eventID, userID string) (int64, error) {
	rank, err := r.client.ZRank(ctx, queueKey(eventID), userID).Result()
	if err == redis.Nil {
		return -1, nil
	}
	if err != nil {
		return 0, fmt.Errorf("zrank: %w", err)
	}
	return rank, nil
}

// PopTop mengambil dan menghapus N user teratas dari sorted set secara atomik via pipeline.
func (r *RedisQueueRepository) PopTop(ctx context.Context, eventID string, n int64) ([]string, error) {
	key := queueKey(eventID)

	// ZPOPMIN adalah atomic pop dari ujung terendah (score terkecil = masuk paling awal).
	results, err := r.client.ZPopMin(ctx, key, n).Result()
	if err != nil {
		return nil, fmt.Errorf("zpopmin: %w", err)
	}

	userIDs := make([]string, len(results))
	for i, z := range results {
		userIDs[i] = z.Member.(string)
	}
	return userIDs, nil
}

func (r *RedisQueueRepository) QueueSize(ctx context.Context, eventID string) (int64, error) {
	return r.client.ZCard(ctx, queueKey(eventID)).Result()
}

func (r *RedisQueueRepository) SetToken(ctx context.Context, eventID, userID, tokenStr string, ttl time.Duration) error {
	return r.client.Set(ctx, tokenKey(eventID, userID), tokenStr, ttl).Err()
}

func (r *RedisQueueRepository) GetToken(ctx context.Context, eventID, userID string) (string, error) {
	val, err := r.client.Get(ctx, tokenKey(eventID, userID)).Result()
	if err == redis.Nil {
		return "", nil
	}
	return val, err
}

func (r *RedisQueueRepository) AddActiveQueue(ctx context.Context, eventID string) error {
	return r.client.SAdd(ctx, activeQueuesKey, eventID).Err()
}

func (r *RedisQueueRepository) GetActiveQueues(ctx context.Context) ([]string, error) {
	return r.client.SMembers(ctx, activeQueuesKey).Result()
}

func (r *RedisQueueRepository) RemoveActiveQueue(ctx context.Context, eventID string) error {
	return r.client.SRem(ctx, activeQueuesKey, eventID).Err()
}

// IncrRequestRate menggunakan sliding window per detik.
// Key expire setelah 2 detik agar tidak menumpuk di Redis.
func (r *RedisQueueRepository) IncrRequestRate(ctx context.Context, eventID string) (int64, error) {
	key := fmt.Sprintf("rate:%s:%d", eventID, time.Now().Unix())

	pipe := r.client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, 2*time.Second)
	if _, err := pipe.Exec(ctx); err != nil {
		return 0, fmt.Errorf("incr rate: %w", err)
	}
	return incrCmd.Val(), nil
}

func queueKey(eventID string) string {
	return "waiting_room:" + eventID
}

func tokenKey(eventID, userID string) string {
	return "queue_token:" + eventID + ":" + userID
}
