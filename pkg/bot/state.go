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
	StateAdminAwaitingCardBank        UserState = "admin_awaiting_card_bank"
	StateAdminAwaitingCardInstructions UserState = "admin_awaiting_card_instructions"
	StateAdminAwaitingStaffAdd        UserState = "admin_awaiting_staff_add"
	StateAdminAwaitingRejectReason    UserState = "admin_awaiting_reject_reason"
	StateAdminAwaitingChannel         UserState = "admin_awaiting_channel"
	StateAdminAwaitingPlanSubscribeID UserState = "admin_awaiting_plan_sub_id"
	StateAdminAwaitingPlanName        UserState = "admin_awaiting_plan_name"
	StateAdminAwaitingPlanPrice       UserState = "admin_awaiting_plan_price"
	StateAdminAwaitingPlanDescription UserState = "admin_awaiting_plan_desc"
	StateAdminAwaitingWelcomeText     UserState = "admin_awaiting_welcome_text"
	StateAdminAwaitingWelcomeImage    UserState = "admin_awaiting_welcome_image"
	StateAdminAwaitingSupportText     UserState = "admin_awaiting_support_text"
	StateAdminAwaitingSupportImage    UserState = "admin_awaiting_support_image"
	StateAdminAwaitingTagDisplayName  UserState = "admin_awaiting_tag_display_name"
	StateAwaitingSubCustomName        UserState = "awaiting_sub_custom_name"
)

type Session struct {
	State               UserState
	PendingAmount       int64
	TempCardNumber      string
	TempCardOwner       string
	TempBankName        string
	TempInstructions    string
	RejectOrderID       int64
	TempPlanSubscribeID int
	TempPlanName        string
	TempPlanPrice       int64
	PurchasingPlanID    int64
	PurchasingPlanName  string
	PurchasingUnitTime  string
}

type SessionManager struct {
	mu       sync.RWMutex
	sessions map[int64]*Session
	inFlight map[int64]bool
}

func NewSessionManager() *SessionManager {
	return &SessionManager{
		sessions: make(map[int64]*Session),
		inFlight: make(map[int64]bool),
	}
}

func (s *SessionManager) TryLock(telegramID int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.inFlight[telegramID] {
		return false
	}
	s.inFlight[telegramID] = true
	return true
}

func (s *SessionManager) Unlock(telegramID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.inFlight, telegramID)
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

func (s *SessionManager) SetTempCardOwner(telegramID int64, cardOwner string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempCardOwner = cardOwner
}

func (s *SessionManager) SetTempBankName(telegramID int64, bankName string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempBankName = bankName
}

func (s *SessionManager) SetTempInstructions(telegramID int64, inst string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.TempInstructions = inst
}

func (s *SessionManager) SetRejectOrderID(telegramID int64, orderID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.RejectOrderID = orderID
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

func (s *SessionManager) SetPurchasingPlan(telegramID int64, planID int64, planName string, unitTime string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sess, exists := s.sessions[telegramID]
	if !exists {
		sess = &Session{}
		s.sessions[telegramID] = sess
	}
	sess.PurchasingPlanID = planID
	sess.PurchasingPlanName = planName
	sess.PurchasingUnitTime = unitTime
}

func (s *SessionManager) Clear(telegramID int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.sessions, telegramID)
}
