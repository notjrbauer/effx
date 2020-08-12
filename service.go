package main

import (
	"log"

	"github.com/go-redis/redis"
)

type word struct {
	Score float64 `json:"score"`
	Key   string  `json:"key"`
}

type service struct {
	cli *redis.Client
	l   *log.Logger
}

func NewService(cli *redis.Client, l *log.Logger) *service {
	return &service{cli, l}
}

func (s *service) TopRanks() ([]word, error) {
	return s.view(0, 9)
}

func (s *service) view(start int64, end int64) ([]word, error) {
	size := end - start
	frequency := make([]word, size+1)

	zRevRangeWithScores := s.cli.ZRevRangeWithScores(leaderboard, start, end)

	for i, data := range zRevRangeWithScores.Val() {
		member, _ := data.Member.(string)

		w := word{Key: member, Score: data.Score}
		frequency[i] = w
	}

	return frequency, nil
}

func (s *service) Standing(id string) ([]word, error) {
	zRank := s.cli.ZRank(leaderboard, id)

	lower := zRank.Val() - 5
	upper := zRank.Val() + 4

	zRangeWithScores := s.cli.ZRangeWithScores(leaderboard, lower, upper)

	frequency := []word{}
	for _, data := range zRangeWithScores.Val() {
		member, _ := data.Member.(string)

		w := word{Key: member, Score: data.Score}
		frequency = append(frequency, w)
	}
	return frequency, nil
}
