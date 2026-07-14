package bot

import (
	"sync"
)

type UserState string

const (
	StateNone                         UserState = "none"
	StateAwaitingAmount               UserState = "awaiting_amount"
	StateAwaitingReceipt              UserState = "awaiting_receipt"
	StateAdminAwaitingCardNumber      UserState = "admin_awaiting_card_number"
	StateAdminAwaitingCardOwner       UserState = "admin_awaiting_card_owner"
	StateAdminAwaitingPlanSubscribeID UserState = "admin_awaiting_plan_sub_id"
	StateAdminAwaitingPlanName        UserState = "admin_awaiting_plan_name"
	StateAdminAwaitingPlanPrice       UserState = "admin_awaiting_plan_price"
	StateAdminAwaitingPlanDescription UserState = "admin_awaiting_plan_desc"
	StateAdminAwaitingWelcomeText     UserState = "admin_awaiting_welcome_text"
	StateAdminAwaitingWelcomeImage    UserState = "admin_awaiting_welcome_image"
)

type Session struct {
	State               UserState
	PendingAmount       int64
	TempCardNumber      string
	TempPlanSubscribeID int
	TempPlanName        string
	TempPlanPrice       int64
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*Session),
	}
}

func (s *SessionManager) Get(telegramID int64) *Session {
	s.mu.RLock()
	defer s.mu.RUnlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		return &Session{State: StateNone}
	}
	return sess
}

func (s *SessionManager) SetState(telegramID int64, state UserState) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.State = state
}

func (s *SessionManager) SetPendingAmount(telegramID int64, amount int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.PendingAmount = amount
}

func (s *SessionManager) SetTempCardNumber(telegramID int64, cardNumber string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempCardNumber = cardNumber
}

func (s *SessionManager) SetTempPlanSubscribeID(telegramID int64, subID int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempPlanSubscribeID = subID
}

func (s *SessionManager) SetTempPlanName(telegramID int64, name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempPlanName = name
}

func (s *SessionManager) SetTempPlanPrice(telegramID int64, price int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempPlanPrice = price
}

func (s *SessionManager) Clear(telegramID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, telegramID)
}
