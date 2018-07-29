package tests

import (
	"github.com/mreider/agilemarkdown/backlog"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestUserSimple(t *testing.T) {
	userContent := `# falconandy
Email: falconandy@yandex.ru

Body`
	user, err := backlog.NewUser(userContent, "test/falconandy.md")
	assert.Nil(t, err)
	assert.Equal(t, "falconandy", user.Name())
	assert.Equal(t, "falconandy@yandex.ru", user.PrimaryEmail())
	assert.Equal(t, []string{"falconandy@yandex.ru"}, user.Emails())
}

func TestUserComplex(t *testing.T) {
	userContent := `# falconandy77
Email: falconandy@yandex.ru, falcon@gmail.com

Body`
	user, err := backlog.NewUser(userContent, "test/falconandy.md")
	assert.Nil(t, err)
	assert.Equal(t, "falconandy77", user.Name())
	assert.Equal(t, "falconandy@yandex.ru", user.PrimaryEmail())
	assert.Equal(t, []string{"falconandy@yandex.ru", "falcon@gmail.com"}, user.Emails())
}

func TestUserAddEmail(t *testing.T) {
	userContent := `# falconandy
Email: falconandy@yandex.ru

Body`
	user, err := backlog.NewUser(userContent, "test/falconandy.md")
	assert.Nil(t, err)
	assert.Equal(t, "falconandy", user.Name())
	assert.Equal(t, "falconandy@yandex.ru", user.PrimaryEmail())
	assert.Equal(t, []string{"falconandy@yandex.ru"}, user.Emails())

	res := user.AddEmailIfNotExist("falcon@gmail.com")
	assert.True(t, res)
	assert.Equal(t, "falconandy", user.Name())
	assert.Equal(t, "falconandy@yandex.ru", user.PrimaryEmail())
	assert.Equal(t, []string{"falconandy@yandex.ru", "falcon@gmail.com"}, user.Emails())

	res = user.AddEmailIfNotExist("falconandy@yandex.ru")
	assert.False(t, res)
	assert.Equal(t, "falconandy", user.Name())
	assert.Equal(t, "falconandy@yandex.ru", user.PrimaryEmail())
	assert.Equal(t, []string{"falconandy@yandex.ru", "falcon@gmail.com"}, user.Emails())
}
