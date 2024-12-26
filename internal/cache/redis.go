package cache

import (
	"context"
	"encoding/json"
	"time"

	"github.com/go-redis/redis/v8"
)

type WalletCache struct {
	client *redis.Client
}

type WalletData struct {
	Balance   float64   `json:"balance"`
	UpdatedAt time.Time `json:"updated_at"`
}

func NewWalletCache(addr string, password string, db int) (*WalletCache, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	if err := client.Ping(context.Background()).Err(); err != nil {
		return nil, err
	}

	return &WalletCache{client: client}, nil
}

func (c *WalletCache) GetWallet(ctx context.Context, walletID string) (*WalletData, error) {
	data, err := c.client.Get(ctx, walletID).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var wallet WalletData
	if err := json.Unmarshal([]byte(data), &wallet); err != nil {
		return nil, err
	}

	return &wallet, nil
}

func (c *WalletCache) SetWallet(ctx context.Context, walletID string, wallet *WalletData, expiration time.Duration) error {
	data, err := json.Marshal(wallet)
	if err != nil {
		return err
	}

	return c.client.Set(ctx, walletID, data, expiration).Err()
}

func (c *WalletCache) InvalidateWallet(ctx context.Context, walletID string) error {
	return c.client.Del(ctx, walletID).Err()
}
