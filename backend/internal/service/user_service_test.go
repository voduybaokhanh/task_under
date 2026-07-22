package service

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestUpdatePushTokenRejectsEmptyToken(t *testing.T) {
	repo := &mockUserRepo{}
	svc := NewUserService(repo)

	assert.Error(t, svc.UpdatePushToken(context.Background(), uuid.New(), ""))
	assert.Empty(t, repo.pushToken, "an empty token must never reach the database")

	assert.NoError(t, svc.UpdatePushToken(context.Background(), uuid.New(), "ExponentPushToken[x]"))
	assert.Equal(t, "ExponentPushToken[x]", repo.pushToken)
}

func TestUpdatePublicKeyRejectsEmptyKey(t *testing.T) {
	repo := &mockUserRepo{}
	svc := NewUserService(repo)

	assert.Error(t, svc.UpdatePublicKey(context.Background(), uuid.New(), ""))
	assert.Empty(t, repo.publicKey)

	assert.NoError(t, svc.UpdatePublicKey(context.Background(), uuid.New(), "dGVzdGtleQ=="))
	assert.Equal(t, "dGVzdGtleQ==", repo.publicKey)
}
