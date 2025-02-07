package repository

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"
	"toyStore/models"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

type SessionRepository interface {
	CreateSession(userId int, role string) (sessionId string, err error)
	CheckSession(sessionId string) (bool, error)
	DeleteSession(sessionId string) (err error)
	RefreshSession(sessionId string, expirationTime time.Duration) (err error)
	GetUserSessionInfo(sessionId string) (userId int, role string, exists bool, err error)
}

type SessionRepo struct {
	rdb *redis.Client
	ctx context.Context
}

func NewSessionRepository(redis_conn *redis.Client, _ctx context.Context) (SessionRepository, error) {
	if redis_conn == nil {
		return nil, errors.New("conn must be non-nil")
	}
	err := redis_conn.Ping(_ctx).Err()
	if err != nil {
		return nil, err
	}
	return &SessionRepo{
		rdb: redis_conn,
		ctx: _ctx,
	}, nil
}

func (s *SessionRepo) CreateSession(userId int, role string) (sessionId string, err error) {
	sessionId = uuid.NewString()
	err = s.rdb.HSet(s.ctx, sessionId, "userId", userId, "role", role).Err()
	if err != nil {
		log.Printf("CreateSession: %v", err)
		err = models.ErrServerError
		return
	}
	expired := 30 * time.Minute
	s.rdb.Expire(s.ctx, sessionId, expired)
	return
}

func (s *SessionRepo) DeleteSession(sessionId string) (err error) {
	err = s.rdb.Del(s.ctx, sessionId).Err()
	if err != nil {
		log.Printf("DeleteSession: %v", err)
		err = models.ErrServerError
	}
	return
}

func (s *SessionRepo) GetUserSessionInfo(sessionId string) (userId int, role string, exists bool, err error) {
	exists, err = s.CheckSession(sessionId)
	if err != nil || !exists {
		return
	}

	val, err := s.rdb.HGetAll(s.ctx, sessionId).Result()
	if err != nil {
		log.Printf("GetUserSessionInfo: %v", err)
		err = models.ErrServerError
		return
	}
	userId, _ = strconv.Atoi(val["userId"])
	role = val["role"]
	exists = true
	return
}

func (s *SessionRepo) CheckSession(sessionId string) (bool, error) {
	exists, err := s.rdb.Exists(s.ctx, sessionId).Result()
	if err != nil {
		log.Printf("CheckSession: %v", err)
		err = models.ErrServerError
		return false, err
	}
	if exists > 0 {
		return true, nil
	}
	return false, nil
}

func (s *SessionRepo) RefreshSession(sessionId string, expirationTime time.Duration) (err error) {
	err = s.rdb.Expire(s.ctx, sessionId, expirationTime).Err()
	if err != nil {
		log.Printf("RefreshSession: %v", err)
		err = models.ErrServerError
	}
	return
}
