package main

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"time"
)

const sessionTTL = 12 * time.Hour

func randHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// hashPassword 盐 + 多次迭代 SHA-256（内部 LAN 工具，零外部依赖）。
func hashPassword(salt, pw string) string {
	h := sha256.Sum256([]byte(salt + pw))
	for i := 0; i < 50000; i++ {
		h = sha256.Sum256(h[:])
	}
	return hex.EncodeToString(h[:])
}

func setPassword(u *User, pw string) {
	u.Salt = randHex(16)
	u.Hash = hashPassword(u.Salt, pw)
}

func checkPassword(u *User, pw string) bool {
	want, _ := hex.DecodeString(u.Hash)
	got, _ := hex.DecodeString(hashPassword(u.Salt, pw))
	return subtle.ConstantTimeCompare(want, got) == 1
}

// ----- 会话 -----

func (s *Store) Login(username, pw string) (string, *User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, u := range s.data.Users {
		if u.Username == username {
			if u.Disabled {
				return "", nil, errors.New("账号已停用")
			}
			if !checkPassword(u, pw) {
				return "", nil, errors.New("用户名或密码错误")
			}
			token := randHex(24)
			s.sessions[token] = session{username: username, expiry: time.Now().Add(sessionTTL)}
			s.audit(username, "登录", "", "")
			_ = s.save()
			return token, u, nil
		}
	}
	return "", nil, errors.New("用户名或密码错误")
}

func (s *Store) Logout(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, token)
}

func (s *Store) UserByToken(token string) (*User, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, ok := s.sessions[token]
	if !ok {
		return nil, false
	}
	if time.Now().After(sess.expiry) {
		delete(s.sessions, token)
		return nil, false
	}
	for _, u := range s.data.Users {
		if u.Username == sess.username && !u.Disabled {
			return u, true
		}
	}
	return nil, false
}

// ----- 用户管理 -----

func (s *Store) Users() []*User {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]*User, len(s.data.Users))
	copy(out, s.data.Users)
	return out
}

func (s *Store) userLocked(username string) *User {
	for _, u := range s.data.Users {
		if u.Username == username {
			return u
		}
	}
	return nil
}

func (s *Store) CreateUser(username, display, role, pw, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if username == "" || pw == "" {
		return errors.New("用户名和密码不能为空")
	}
	if s.userLocked(username) != nil {
		return errors.New("用户名已存在")
	}
	u := &User{Username: username, Display: display, Role: role}
	setPassword(u, pw)
	s.data.Users = append(s.data.Users, u)
	s.audit(actor, "新建账号", username, roleLabel(role))
	return s.save()
}

func (s *Store) UpdateUser(username, display, role string, disabled bool, newPw, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.userLocked(username)
	if u == nil {
		return errors.New("账号不存在")
	}
	u.Display, u.Role, u.Disabled = display, role, disabled
	if newPw != "" {
		setPassword(u, newPw)
	}
	s.audit(actor, "修改账号", username, roleLabel(role))
	return s.save()
}

func (s *Store) DeleteUser(username, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if username == actor {
		return errors.New("不能删除当前登录的账号")
	}
	for i, u := range s.data.Users {
		if u.Username == username {
			s.data.Users = append(s.data.Users[:i], s.data.Users[i+1:]...)
			s.audit(actor, "删除账号", username, "")
			return s.save()
		}
	}
	return errors.New("账号不存在")
}

// ChangeOwnPassword 用户改自己的密码。
func (s *Store) ChangeOwnPassword(username, oldPw, newPw string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := s.userLocked(username)
	if u == nil {
		return errors.New("账号不存在")
	}
	if !checkPassword(u, oldPw) {
		return errors.New("原密码错误")
	}
	if newPw == "" {
		return errors.New("新密码不能为空")
	}
	setPassword(u, newPw)
	s.audit(username, "修改本人密码", username, "")
	return s.save()
}
