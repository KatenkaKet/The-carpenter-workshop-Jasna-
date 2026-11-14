package main

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
	"runtime"
	"syscall"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const dbFile = "data.json"

type Child struct {
	Name string `json:"name"`
	Age  int    `json:"age"`
}

type Family struct {
	ID                     int     `json:"id"`
	ParentName             string  `json:"parent_name"`
	Phone                  string  `json:"phone"`
	Email                  string  `json:"email"`
	Token                  string  `json:"token"`
	LessonsRemaining       int     `json:"lessons_remaining"`
	SubscriptionValidUntil string  `json:"subscription_valid_until"`
	MoneyBalanceRub        int     `json:"money_balance_rub"`
	FreeTrialUsed          bool    `json:"free_trial_used"`
	Children               []Child `json:"children"`
}

type Attendance struct {
	ID            int    `json:"id"`
	FamilyID      int    `json:"family_id"`
	DateFormatted string `json:"date_formatted"`
	ChildName     string `json:"child_name"`
	StatusLabel   string `json:"status_label"`
}

type Payment struct {
	ID             int    `json:"id"`
	FamilyID       int    `json:"family_id"`
	ParentName     string `json:"parent_name"`
	DateFormatted  string `json:"date_formatted"`
	AmountRub      int    `json:"amount_rub"`
	StatusLabel    string `json:"status_label"`
	IdempotencyKey string `json:"idempotency_key"`
	ReceiptURL     string `json:"receipt_url"`
}

type Booking struct {
	ID               int    `json:"id"`
	Date             string `json:"date"`
	TimeSlot         string `json:"time_slot"`
	LessonDate       string `json:"lesson_date"`
	FamilyID         int    `json:"family_id"`
	ChildName        string `json:"child_name"`
	ParentName       string `json:"parent_name"`
	LessonsRemaining int    `json:"lessons_remaining"`
	IsFreeTrial      bool   `json:"is_free_trial"`
}

type Notification struct {
	ID       int    `json:"id"`
	FamilyID int    `json:"family_id"`
	Date     string `json:"date"`
	Text     string `json:"text"`
	IsRead   bool   `json:"is_read"`
}

type AuditLog struct {
	Time    string `json:"time"`
	Message string `json:"message"`
}

type OTPChallenge struct {
	Role      string
	FamilyID  int
	Contact   string
	Code      string
	ExpiresAt time.Time
}

type AppState struct {
	NextID           int             `json:"next_id"`
	MasterSession    string          `json:"master_session"`
	LogoURL          string          `json:"logo_url"`
	HeroImageURL     string          `json:"hero_image_url"`
	GroupCapacity    int             `json:"group_capacity"`
	SubscriptionDays int             `json:"subscription_days"`
	FreeTrialEnabled bool            `json:"free_trial_enabled"`
	WorkHours        string          `json:"work_hours"`
	DaysOff          string          `json:"days_off"`
	ClosureActive    bool            `json:"closure_active"`
	ClosureNotice    string          `json:"closure_notice"`
	PauseBooking     bool            `json:"pause_booking"`
	ClosedSlots      map[string]bool `json:"closed_slots"`
	Families         []Family        `json:"families"`
	Bookings         []Booking       `json:"bookings"`
	Attendances      []Attendance    `json:"attendances"`
	Payments         []Payment       `json:"payments"`
	Notifications    []Notification  `json:"notifications"`
	AuditLogs        []AuditLog      `json:"audit_logs"`
}

type BasePage struct {
	PageTitle string
	NoIndex   bool
	LogoURL   string
}

type LandingData struct {
	BasePage
	FlashMessage     string
	ErrorMessage     string
	HeroImageURL     string
	FreeTrialEnabled bool
	WorkHours        string
	DaysOff          string
	ClosureActive    bool
	ClosureNotice    string
	PauseBooking     bool
	DefaultDate      string
	GroupCapacity    int
	SubDays          int
	ClosedSlotsJSON  template.JS
	BookedCountsJSON template.JS
}

type LoginData struct {
	BasePage
	Role         string
	Step         string
	Contact      string
	DemoCode     string
	ErrorMessage string
}

type CabinetData struct {
	BasePage
	Token                    string
	FlashMessage             string
	ErrorMessage             string
	NextIdemKey4             string
	NextIdemKey1             string
	SubWarning               string
	UnreadNotificationsCount int
	Family                   Family
	Children                 []Child
	ActiveBookings           []Booking
	Attendances              []Attendance
	Payments                 []Payment
	Notifications            []Notification
}

type MasterData struct {
	BasePage
	FlashMessage     string
	DefaultDate      string
	GroupCapacity    int
	FreeTrialEnabled bool
	WorkHours        string
	DaysOff          string
	ClosureActive    bool
	ClosureNotice    string
	PauseBooking     bool
	SubDays          int
	ExpiringFamilies []string
	ClosedSlotsList  []string
	Bookings         []Booking
	Families         []Family
	PendingReceipts  []Payment
	AuditLogs        []AuditLog
}

type LegalData struct {
	BasePage
	DocTitle string
	DocText  string
}

var (
	mu         sync.Mutex
	state      AppState
	challenges = make(map[string]OTPChallenge)
)

func genToken() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

func genOTP() string {
	n, _ := rand.Int(rand.Reader, big.NewInt(900000))
	return fmt.Sprintf("%06d", n.Int64()+100000)
}

func normalizePhone(s string) string {
	return strings.Map(func(c rune) rune {
		if c >= '0' && c <= '9' {
			return c
		}
		return -1
	}, s)
}

func initDemoScenarioLocked() {
	state = AppState{
		NextID:           200,
		MasterSession:    genToken(),
		LogoURL:          state.LogoURL,
		HeroImageURL:     "https://unsplash.com",
		GroupCapacity:    2,
		SubscriptionDays: 35,
		FreeTrialEnabled: true,
		WorkHours:        "Ежедневно: 10:00–12:00, 13:00–15:00, 16:00–18:00",
		DaysOff:          "Без выходных (по предварительной записи)",
		ClosureNotice:    "Внимание! Мастерская временно не работает в связи с подготовкой новых заготовок.",
		ClosedSlots:      make(map[string]bool),
		Families: []Family{{
			ID: 1, ParentName: "Елена Смирнова", Phone: "+7 (912) 000-00-01", Email: "elena@example.ru",
			Token: "demo-family-token-9f8a7b6c5d4e3f2a", LessonsRemaining: 0, FreeTrialUsed: false,
			Children: []Child{{Name: "Артём", Age: 8}, {Name: "Софья", Age: 11}},
		}},
		Bookings:      []Booking{},
		Attendances:   []Attendance{},
		Payments:      []Payment{},
		Notifications: []Notification{{ID: 1, FamilyID: 1, Date: time.Now().Format("02.01.2006"), Text: "Добро пожаловать в кабинет семьи Смирновых (2 ребёнка, группа на 2 места)."}},
		AuditLogs:     []AuditLog{{Time: time.Now().Format("02.01 15:04"), Message: "Загружен сценарий приёмки: группа на 2 места, семья с двумя детьми."}},
	}
	saveStateLocked()
}

func loadOrInitState() {
	_ = os.MkdirAll("uploads", 0755)
	data, err := os.ReadFile(dbFile)
	if err == nil && json.Unmarshal(data, &state) == nil && state.GroupCapacity > 0 {
		if state.ClosedSlots == nil {
			state.ClosedSlots = make(map[string]bool)
		}
		if state.MasterSession == "" {
			state.MasterSession = genToken()
		}
		return
	}
	initDemoScenarioLocked()
}

func saveStateLocked() {
	if bytes, err := json.MarshalIndent(state, "", "  "); err == nil {
		_ = os.WriteFile(dbFile, bytes, 0644)
	}
}

func addAuditLocked(msg string) {
	state.AuditLogs = append([]AuditLog{{Time: time.Now().Format("02.01 15:04"), Message: msg}}, state.AuditLogs...)
	if len(state.AuditLogs) > 30 {
		state.AuditLogs = state.AuditLogs[:30]
	}
}

func addNotifLocked(familyID int, text string) {
	state.NextID++
	state.Notifications = append([]Notification{{ID: state.NextID, FamilyID: familyID, Date: time.Now().Format("02.01.2006"), Text: text}}, state.Notifications...)
}

func findFamilyLocked(id int) *Family {
	for i := range state.Families {
		if state.Families[i].ID == id {
			return &state.Families[i]
		}
	}
	return nil
}

func renderPage(w http.ResponseWriter, file string, data any) {
	tmpl, err := template.ParseFiles("templates/layout.html", "templates/"+file)
	if err != nil {
		http.Error(w, "Ошибка шаблона: "+err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = tmpl.ExecuteTemplate(w, "layout", data)
}

func isMasterAuth(r *http.Request) bool {
	c, err := r.Cookie("yasna_master_auth")
	if err != nil {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	return c.Value != "" && c.Value == state.MasterSession
}


func checkSubscriptionWarning(f Family) string {
	var reasons []string
	if f.SubscriptionValidUntil != "" {
		if exp, err := time.Parse("02.01.2006", f.SubscriptionValidUntil); err == nil {
			daysLeft := int(time.Until(exp).Hours()/24) + 1
			if daysLeft < 0 {
				reasons = append(reasons, fmt.Sprintf("срок действия абонемента истёк %s", f.SubscriptionValidUntil))
			} else if daysLeft <= 5 {
				reasons = append(reasons, fmt.Sprintf("срок действия абонемента заканчивается %s (осталось дн.: %d)", f.SubscriptionValidUntil, daysLeft))
			}
		}
	}
	if f.LessonsRemaining == 1 && f.SubscriptionValidUntil != "" {
		reasons = append(reasons, "осталось последнее (1) занятие по абонементу")
	} else if f.LessonsRemaining == 0 && f.SubscriptionValidUntil != "" {
		reasons = append(reasons, "занятия по абонементу закончились (остаток: 0)")
	}
	if len(reasons) == 0 {
		return ""
	}
	return "Внимание! По абонементу семьи " + f.ParentName + " (" + f.Phone + "): " + strings.Join(reasons, "; ") + "."
}

func handleHome(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	mu.Lock()
	flash, errMsg := "", ""
	switch r.URL.Query().Get("status") {
	case "booked":
		flash = "Запись подтверждена! Свободные места в смене обновлены."
	case "booked_trial":
		flash = "Запись на бесплатный пробный мастер-класс (0 ₽) подтверждена!"
	case "no_seats":
		errMsg = "Отказ: в выбранной смене недостаточно свободных мест (защита от переполнения группы)."
	case "slot_closed":
		errMsg = "Выбранное время закрыто мастером для записи."
	case "duplicate":
		errMsg = "Ребёнок уже записан на эту дату и смену."
	case "paused":
		errMsg = "Онлайн-запись временно приостановлена в связи с объявлением мастерской."
	}
	bookedCounts := make(map[string]int)
	for _, b := range state.Bookings {
		bookedCounts[b.Date+"|"+b.TimeSlot]++
	}
	cJSON, _ := json.Marshal(state.ClosedSlots)
	bJSON, _ := json.Marshal(bookedCounts)

	data := LandingData{
		BasePage: BasePage{PageTitle: "Запись на занятия", LogoURL: state.LogoURL}, FlashMessage: flash, ErrorMessage: errMsg,
		HeroImageURL: state.HeroImageURL, FreeTrialEnabled: state.FreeTrialEnabled, WorkHours: state.WorkHours,
		DaysOff: state.DaysOff, ClosureActive: state.ClosureActive, ClosureNotice: state.ClosureNotice, PauseBooking: state.PauseBooking,
		DefaultDate: time.Now().AddDate(0, 0, 1).Format("2006-01-02"), GroupCapacity: state.GroupCapacity, SubDays: state.SubscriptionDays,
		ClosedSlotsJSON: template.JS(cJSON), BookedCountsJSON: template.JS(bJSON),
	}
	mu.Unlock()
	renderPage(w, "landing.html", data)
}

func handleBook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	mu.Lock()
	if state.ClosureActive && state.PauseBooking {
		mu.Unlock()
		http.Redirect(w, r, "/?status=paused#schedule", http.StatusSeeOther)
		return
	}
	mu.Unlock()

	_ = r.ParseForm()
	dateStr := strings.TrimSpace(r.FormValue("date"))
	timeSlot := strings.TrimSpace(r.FormValue("time_slot"))
	parentName := strings.TrimSpace(r.FormValue("parent_name"))
	phone := strings.TrimSpace(r.FormValue("phone"))
	email := strings.TrimSpace(r.FormValue("email"))
	wantTrial := r.FormValue("is_free_trial") == "1"
	cNames, cAges := r.Form["child_name[]"], r.Form["child_age[]"]

	var kids []Child
	for i, rn := range cNames {
		if n := strings.TrimSpace(rn); n != "" {
			age := 0
			if i < len(cAges) {
				age, _ = strconv.Atoi(strings.TrimSpace(cAges[i]))
			}
			kids = append(kids, Child{Name: n, Age: age})
		}
	}
	if len(kids) == 0 || parentName == "" || dateStr == "" || timeSlot == "" {
		http.Redirect(w, r, "/#schedule", http.StatusSeeOther)
		return
	}

	mu.Lock()
	if state.ClosedSlots[dateStr+"|"+timeSlot] {
		mu.Unlock()
		http.Redirect(w, r, "/?status=slot_closed#schedule", http.StatusSeeOther)
		return
	}
	bookedNow := 0
	for _, b := range state.Bookings {
		if b.Date == dateStr && b.TimeSlot == timeSlot {
			bookedNow++
		}
	}
	if state.GroupCapacity-bookedNow < len(kids) {
		mu.Unlock()
		http.Redirect(w, r, "/?status=no_seats#schedule", http.StatusSeeOther)
		return
	}

	var fam *Family
	pNorm := normalizePhone(phone)
	for i := range state.Families {
		if (pNorm != "" && normalizePhone(state.Families[i].Phone) == pNorm) || (email != "" && strings.EqualFold(state.Families[i].Email, email)) {
			fam = &state.Families[i]
			break
		}
	}
	if fam == nil {
		state.NextID++
		state.Families = append(state.Families, Family{ID: state.NextID, ParentName: parentName, Phone: phone, Email: email, Token: genToken(), Children: kids})
		fam = &state.Families[len(state.Families)-1]
	} else {
		if email != "" && fam.Email == "" {
			fam.Email = email
		}
		for _, k := range kids {
			ex := false
			for _, c := range fam.Children {
				if strings.EqualFold(c.Name, k.Name) {
					ex = true
					break
				}
			}
			if !ex {
				fam.Children = append(fam.Children, k)
			}
		}
	}

	isTrial := wantTrial && state.FreeTrialEnabled && !fam.FreeTrialUsed
	if isTrial {
		fam.FreeTrialUsed = true
	}

	count := 0
	lDate := dateStr + " (" + timeSlot + ")"
	for _, k := range kids {
		dup := false
		for _, b := range state.Bookings {
			if b.Date == dateStr && b.TimeSlot == timeSlot && b.FamilyID == fam.ID && strings.EqualFold(b.ChildName, k.Name) {
				dup = true
				break
			}
		}
		if dup {
			continue
		}
		state.NextID++
		state.Bookings = append(state.Bookings, Booking{
			ID: state.NextID, Date: dateStr, TimeSlot: timeSlot, LessonDate: lDate,
			FamilyID: fam.ID, ChildName: k.Name, ParentName: fam.ParentName, LessonsRemaining: fam.LessonsRemaining, IsFreeTrial: isTrial,
		})
		addNotifLocked(fam.ID, fmt.Sprintf("Запись подтверждена: %s на %s.", k.Name, lDate))
		addAuditLocked(fmt.Sprintf("Запись: %s (%s) на %s", k.Name, fam.ParentName, lDate))
		count++
	}
	saveStateLocked()
	mu.Unlock()

	if count == 0 {
		http.Redirect(w, r, "/?status=duplicate#schedule", http.StatusSeeOther)
	} else if isTrial {
		http.Redirect(w, r, "/?status=booked_trial#schedule", http.StatusSeeOther)
	} else {
		http.Redirect(w, r, "/?status=booked#schedule", http.StatusSeeOther)
	}
}

func handleAuth(w http.ResponseWriter, r *http.Request) {
	role := "parent"
	pageTitle := "Вход для родителей"
	if strings.HasPrefix(r.URL.Path, "/master-login") {
		role = "master"
		pageTitle = "Вход для мастера"
	}

	mu.Lock()
	logo := state.LogoURL
	if r.Method == http.MethodGet {
		mu.Unlock()
		renderPage(w, "login.html", LoginData{BasePage: BasePage{PageTitle: pageTitle, NoIndex: true, LogoURL: logo}, Role: role, Step: "request"})
		return
	}

	step := r.FormValue("step")
	contact := strings.TrimSpace(r.FormValue("contact"))

	if step == "request" {
		if role == "master" {
			cNorm := normalizePhone(contact)
			if !strings.EqualFold(contact, "master@yasna.ru") && cNorm != "79124120005" && cNorm != "89124120005" {
				mu.Unlock()
				renderPage(w, "login.html", LoginData{BasePage: BasePage{PageTitle: pageTitle, NoIndex: true, LogoURL: logo}, Role: role, Step: "request", ErrorMessage: "Укажите Email (master@yasna.ru) или телефон (+79124120005) мастера."})
				return
			}
			code := genOTP()
			challenges["master|"+contact] = OTPChallenge{Role: "master", Contact: contact, Code: code, ExpiresAt: time.Now().Add(10 * time.Minute)}
			addAuditLocked(fmt.Sprintf("Отправлен код подтверждения входа мастера на %s", contact))
			saveStateLocked()
			mu.Unlock()
			renderPage(w, "login.html", LoginData{BasePage: BasePage{PageTitle: pageTitle, NoIndex: true, LogoURL: logo}, Role: role, Step: "verify", Contact: contact, DemoCode: code})
			return
		}

		cNorm := normalizePhone(contact)
		var found *Family
		for i := range state.Families {
			if (cNorm != "" && len(cNorm) >= 6 && strings.HasSuffix(normalizePhone(state.Families[i].Phone), cNorm)) || (contact != "" && strings.EqualFold(state.Families[i].Email, contact)) {
				found = &state.Families[i]
				break
			}
		}
		if found == nil {
			mu.Unlock()
			renderPage(w, "login.html", LoginData{BasePage: BasePage{PageTitle: pageTitle, NoIndex: true, LogoURL: logo}, Role: role, Step: "request", ErrorMessage: "Семья с таким телефоном или Email не найдена. Сначала запишите ребёнка на занятие."})
			return
		}
		code := genOTP()
		challenges["parent|"+contact] = OTPChallenge{Role: "parent", FamilyID: found.ID, Contact: contact, Code: code, ExpiresAt: time.Now().Add(10 * time.Minute)}
		mu.Unlock()
		renderPage(w, "login.html", LoginData{BasePage: BasePage{PageTitle: pageTitle, NoIndex: true, LogoURL: logo}, Role: role, Step: "verify", Contact: contact, DemoCode: code})
		return
	}

	if step == "verify" {
		code := strings.TrimSpace(r.FormValue("code"))
		ch, ok := challenges[role+"|"+contact]
		if !ok || time.Now().After(ch.ExpiresAt) || ch.Code != code {
			mu.Unlock()
			renderPage(w, "login.html", LoginData{BasePage: BasePage{PageTitle: pageTitle, NoIndex: true, LogoURL: logo}, Role: role, Step: "verify", Contact: contact, DemoCode: ch.Code, ErrorMessage: "Неверный или просроченный код подтверждения."})
			return
		}
		delete(challenges, role+"|"+contact)
		if role == "master" {
			if state.MasterSession == "" {
				state.MasterSession = genToken()
				saveStateLocked()
			}
			sess := state.MasterSession
			addAuditLocked(fmt.Sprintf("Успешный вход мастера по одноразовому коду (%s)", contact))
			saveStateLocked()
			mu.Unlock()
			http.SetCookie(w, &http.Cookie{Name: "yasna_master_auth", Value: sess, Path: "/", HttpOnly: true})
			http.Redirect(w, r, "/master", http.StatusSeeOther)
			return
		}
		fam := findFamilyLocked(ch.FamilyID)
		if fam == nil {
			mu.Unlock()
			http.Redirect(w, r, "/parent-login", http.StatusSeeOther)
			return
		}
		tok := fam.Token
		mu.Unlock()
		http.Redirect(w, r, "/cabinet/"+tok, http.StatusSeeOther)
		return
	}
	mu.Unlock()
}

func handleCabinet(w http.ResponseWriter, r *http.Request) {
	trimmed := strings.TrimPrefix(r.URL.Path, "/cabinet/")
	token, action, hasAction := strings.Cut(trimmed, "/")

	mu.Lock()
	var found *Family
	for i := range state.Families {
		if state.Families[i].Token == token {
			found = &state.Families[i]
			break
		}
	}
	if found == nil {
		mu.Unlock()
		http.Redirect(w, r, "/parent-login", http.StatusSeeOther)
		return
	}

	flash, errMsg := "", ""
	switch r.URL.Query().Get("msg") {
	case "paid":
		flash = "Оплата успешно зачислена! Баланс занятий обновлён."
	case "dup_pay":
		errMsg = "Защита от дублирования: платёж с таким идентификатором уже был зачислен ранее. Повторное начисление заблокировано."
	}

	if r.Method == http.MethodPost && hasAction {
		switch action {
		case "pay":
			idemKey := strings.TrimSpace(r.FormValue("idem_key"))
			for _, p := range state.Payments {
				if p.IdempotencyKey == idemKey {
					addAuditLocked(fmt.Sprintf("Заблокирован дубль платежа (%s) для %s", idemKey, found.ParentName))
					saveStateLocked()
					mu.Unlock()
					http.Redirect(w, r, "/cabinet/"+token+"?msg=dup_pay", http.StatusSeeOther)
					return
				}
			}
			amount, addL := 900, 1
			if r.FormValue("tariff") == "sub_4" {
				amount, addL = 3200, 4
				days := state.SubscriptionDays
				if days <= 0 { days = 35 }
				found.SubscriptionValidUntil = time.Now().AddDate(0, 0, days).Format("02.01.2006")
			}
			found.LessonsRemaining += addL
			state.NextID++
			state.Payments = append([]Payment{{
				ID: state.NextID, FamilyID: found.ID, ParentName: found.ParentName, DateFormatted: time.Now().Format("02.01.2006"),
				AmountRub: amount, StatusLabel: "Оплачено онлайн (СБП)", IdempotencyKey: idemKey,
			}}, state.Payments...)
			addNotifLocked(found.ID, fmt.Sprintf("Оплата %d ₽ получена (+%d зан.).", amount, addL))
			addAuditLocked(fmt.Sprintf("Онлайн-оплата %d ₽ от %s (+%d ур., ID: %s)", amount, found.ParentName, addL, idemKey))
			saveStateLocked()
			mu.Unlock()
			http.Redirect(w, r, "/cabinet/"+token+"?msg=paid", http.StatusSeeOther)
			return
		case "cancel-booking":
			bID, _ := strconv.Atoi(r.FormValue("booking_id"))
			for i := range state.Bookings {
				if state.Bookings[i].ID == bID && state.Bookings[i].FamilyID == found.ID {
					b := state.Bookings[i]
					if b.IsFreeTrial {
						found.FreeTrialUsed = false
					}
					state.NextID++
					state.Attendances = append([]Attendance{{ID: state.NextID, FamilyID: found.ID, DateFormatted: time.Now().Format("02.01.2006"), ChildName: b.ChildName, StatusLabel: "Отменено родителем"}}, state.Attendances...)
					addNotifLocked(found.ID, fmt.Sprintf("Вы отменили запись (%s) на %s.", b.ChildName, b.LessonDate))
					addAuditLocked(fmt.Sprintf("Родитель %s отменил запись (%s)", found.ParentName, b.ChildName))
					state.Bookings = append(state.Bookings[:i], state.Bookings[i+1:]...)
					saveStateLocked()
					break
				}
			}
		case "read-notifications":
			for i := range state.Notifications {
				if state.Notifications[i].FamilyID == found.ID {
					state.Notifications[i].IsRead = true
				}
			}
			saveStateLocked()
		}
		mu.Unlock()
		http.Redirect(w, r, "/cabinet/"+token, http.StatusSeeOther)
		return
	}

	famCopy := *found
	logo := state.LogoURL
	var famB []Booking
	for _, b := range state.Bookings {
		if b.FamilyID == found.ID {
			famB = append(famB, b)
		}
	}
	var famA []Attendance
	for _, a := range state.Attendances {
		if a.FamilyID == found.ID {
			famA = append(famA, a)
		}
	}
	var famP []Payment
	for _, p := range state.Payments {
		if p.FamilyID == found.ID {
			famP = append(famP, p)
		}
	}
	var famN []Notification
	unread := 0
	for _, n := range state.Notifications {
		if n.FamilyID == found.ID {
			famN = append(famN, n)
			if !n.IsRead {
				unread++
			}
		}
	}
	mu.Unlock()

	renderPage(w, "cabinet.html", CabinetData{
		BasePage: BasePage{PageTitle: "Личный кабинет", NoIndex: true, LogoURL: logo}, Token: famCopy.Token,
		FlashMessage: flash, ErrorMessage: errMsg, NextIdemKey4: "pay_" + genToken()[:8], NextIdemKey1: "pay_" + genToken()[:8],
		UnreadNotificationsCount: unread, Family: famCopy, Children: famCopy.Children, ActiveBookings: famB, Attendances: famA, Payments: famP, Notifications: famN,
	})
}

func handleMaster(w http.ResponseWriter, r *http.Request) {
	if !isMasterAuth(r) {
		http.Redirect(w, r, "/master-login", http.StatusSeeOther)
		return
	}
	mu.Lock()
	flash := ""
	if r.URL.Query().Get("msg") == "demo_reset" {
		flash = "Сценарий приёмки загружен: группа на 2 места и семья Смирновых с двумя детьми (Артём и Софья)."
	}
	for i := range state.Bookings {
		if fam := findFamilyLocked(state.Bookings[i].FamilyID); fam != nil {
			state.Bookings[i].LessonsRemaining = fam.LessonsRemaining
		}
	}
	var pending []Payment
	for _, p := range state.Payments {
		if p.ReceiptURL == "" {
			pending = append(pending, p)
		}
	}
	var expiring []string
	for _, f := range state.Families {
		if w := checkSubscriptionWarning(f); w != "" {
			expiring = append(expiring, w)
		}
	}
	var closedList []string
	for k, v := range state.ClosedSlots {
		if v {
			closedList = append(closedList, strings.ReplaceAll(k, "|", " • "))
		}
	}
	sort.Strings(closedList)

	data := MasterData{
		BasePage: BasePage{PageTitle: "Панель мастера", NoIndex: true, LogoURL: state.LogoURL}, FlashMessage: flash,
		DefaultDate: time.Now().AddDate(0, 0, 1).Format("2006-01-02"), GroupCapacity: state.GroupCapacity,
		FreeTrialEnabled: state.FreeTrialEnabled, WorkHours: state.WorkHours, DaysOff: state.DaysOff,
		ClosureActive: state.ClosureActive, ClosureNotice: state.ClosureNotice, PauseBooking: state.PauseBooking,
		ClosedSlotsList: closedList, Bookings: append([]Booking(nil), state.Bookings...),
		Families: append([]Family(nil), state.Families...), PendingReceipts: pending, AuditLogs: append([]AuditLog(nil), state.AuditLogs...),
	}
	mu.Unlock()
	renderPage(w, "master.html", data)
}

func saveUploadedFile(r *http.Request, field string) string {
	file, header, err := r.FormFile(field)
	if err != nil {
		return ""
	}
	defer file.Close()
	ext := filepath.Ext(header.Filename)
	if ext == "" {
		ext = ".jpg"
	}
	fn := fmt.Sprintf("%s_%s%s", field, genToken()[:8], ext)
	dst, err := os.Create(filepath.Join("uploads", fn))
	if err != nil {
		return ""
	}
	defer dst.Close()
	_, _ = io.Copy(dst, file)
	return "/uploads/" + fn
}

func handleMasterPost(w http.ResponseWriter, r *http.Request) {
	if !isMasterAuth(r) || r.Method != http.MethodPost {
		http.Redirect(w, r, "/master-login", http.StatusSeeOther)
		return
	}
	if r.URL.Path == "/master/logout" {
		http.SetCookie(w, &http.Cookie{Name: "yasna_master_auth", Value: "", Path: "/", MaxAge: -1})
		http.Redirect(w, r, "/", http.StatusSeeOther)
		return
	}
	if r.URL.Path == "/master/demo-reset" {
		mu.Lock()
		initDemoScenarioLocked()
		mu.Unlock()
		http.Redirect(w, r, "/master?msg=demo_reset", http.StatusSeeOther)
		return
	}
	if r.URL.Path == "/master/media/upload" {
		_ = r.ParseMultipartForm(10 << 20)
		nl, nh := saveUploadedFile(r, "logo_file"), saveUploadedFile(r, "hero_file")
		mu.Lock()
		if nl != "" {
			state.LogoURL = nl
			addAuditLocked("Мастер обновил логотип сайта")
		}
		if nh != "" {
			state.HeroImageURL = nh
			addAuditLocked("Мастер обновил фото обложки сайта")
		}
		saveStateLocked()
		mu.Unlock()
		http.Redirect(w, r, "/master", http.StatusSeeOther)
		return
	}

	mu.Lock()
	switch r.URL.Path {
	case "/master/slots/toggle":
		d, s := strings.TrimSpace(r.FormValue("date")), strings.TrimSpace(r.FormValue("time_slot"))
		if d != "" && s != "" {
			key := d + "|" + s
			if state.ClosedSlots[key] {
				delete(state.ClosedSlots, key)
				addAuditLocked(fmt.Sprintf("Открыта смена %s (%s)", d, s))
			} else {
				state.ClosedSlots[key] = true
				addAuditLocked(fmt.Sprintf("Закрыта смена %s (%s)", d, s))
			}
		}
	case "/master/settings/schedule":
		if wh := strings.TrimSpace(r.FormValue("work_hours")); wh != "" {
			state.WorkHours = wh
		}
		if do := strings.TrimSpace(r.FormValue("days_off")); do != "" {
			state.DaysOff = do
		}
		state.ClosureActive = r.FormValue("closure_active") == "1"
		state.PauseBooking = r.FormValue("pause_booking") == "1"
		if cn := strings.TrimSpace(r.FormValue("closure_notice")); cn != "" {
			state.ClosureNotice = cn
		}
		addAuditLocked("Обновлён график работы / объявление")
	case "/master/settings/sub-days":
		if d, _ := strconv.Atoi(r.FormValue("sub_days")); d >= 7 && d <= 365 {
			state.SubscriptionDays = d
			addAuditLocked(fmt.Sprintf("Мастер установил срок действия абонемента: %d дней", d))
		}
	case "/master/settings/capacity":
		if c, _ := strconv.Atoi(r.FormValue("capacity")); c >= 1 && c <= 20 {
			state.GroupCapacity = c
			addAuditLocked(fmt.Sprintf("Вместимость смены изменена на %d мест", c))
		}
	case "/master/toggle-free-trial":
		state.FreeTrialEnabled = !state.FreeTrialEnabled
		addAuditLocked("Переключён режим бесплатного первого МК")

	case "/master/payments/manual":
		fID, _ := strconv.Atoi(r.FormValue("family_id"))
		amt, _ := strconv.Atoi(r.FormValue("amount_rub"))
		addL, _ := strconv.Atoi(r.FormValue("lessons_add"))
		rURL := strings.TrimSpace(r.FormValue("receipt_url"))
		if fam := findFamilyLocked(fID); fam != nil && addL > 0 {
			fam.LessonsRemaining += addL
			state.NextID++
			state.Payments = append([]Payment{{ID: state.NextID, FamilyID: fam.ID, ParentName: fam.ParentName, DateFormatted: time.Now().Format("02.01.2006"), AmountRub: amt, StatusLabel: "Оплачено мастеру", IdempotencyKey: "man_" + genToken()[:6], ReceiptURL: rURL}}, state.Payments...)
			addNotifLocked(fam.ID, fmt.Sprintf("Мастер зачислил оплату %d ₽ (+%d зан.).", amt, addL))
			addAuditLocked(fmt.Sprintf("Ручная оплата %d ₽ для %s (+%d ур.)", amt, fam.ParentName, addL))
		}
	case "/master/payments/attach-receipt":
		pID, _ := strconv.Atoi(r.FormValue("payment_id"))
		rURL := strings.TrimSpace(r.FormValue("receipt_url"))
		for i := range state.Payments {
			if state.Payments[i].ID == pID && rURL != "" {
				state.Payments[i].ReceiptURL = rURL
				addNotifLocked(state.Payments[i].FamilyID, fmt.Sprintf("Прикреплён чек ФНС «Мой налог» на %d ₽.", state.Payments[i].AmountRub))
				addAuditLocked(fmt.Sprintf("Прикреплён чек к платежу %d ₽ (%s)", state.Payments[i].AmountRub, state.Payments[i].ParentName))
				break
			}
		}
	case "/master/attendance":
		bID, _ := strconv.Atoi(r.FormValue("booking_id"))
		st := r.FormValue("status")
		idx := -1
		for i := range state.Bookings {
			if state.Bookings[i].ID == bID {
				idx = i
				break
			}
		}
		if idx != -1 {
			b := state.Bookings[idx]
			fam := findFamilyLocked(b.FamilyID)
			today := time.Now().Format("02.01.2006")
			state.NextID++
			switch st {
			case "visited":
				if b.IsFreeTrial {
					state.Attendances = append([]Attendance{{ID: state.NextID, FamilyID: b.FamilyID, DateFormatted: today, ChildName: b.ChildName, StatusLabel: "Посетил пробный МК (0 ₽)"}}, state.Attendances...)
					if fam != nil {
						addNotifLocked(fam.ID, fmt.Sprintf("Ребёнок (%s) посетил бесплатный мастер-класс!", b.ChildName))
					}
					addAuditLocked(fmt.Sprintf("Посетил пробный МК: %s (%s)", b.ChildName, b.ParentName))
				} else {
					if fam != nil && fam.LessonsRemaining > 0 {
						fam.LessonsRemaining--
					}
					state.Attendances = append([]Attendance{{ID: state.NextID, FamilyID: b.FamilyID, DateFormatted: today, ChildName: b.ChildName, StatusLabel: "Посетил (−1 урок)"}}, state.Attendances...)
					if fam != nil {
						addNotifLocked(fam.ID, fmt.Sprintf("Отмечено посещение (%s). Остаток: %d.", b.ChildName, fam.LessonsRemaining))
					}
					addAuditLocked(fmt.Sprintf("Посетил: %s (%s), остаток: %d", b.ChildName, b.ParentName, fam.LessonsRemaining))
				}
			case "missed":
				if !b.IsFreeTrial && fam != nil && fam.LessonsRemaining > 0 {
					fam.LessonsRemaining--
				}
				state.Attendances = append([]Attendance{{ID: state.NextID, FamilyID: b.FamilyID, DateFormatted: today, ChildName: b.ChildName, StatusLabel: "Пропустил"}}, state.Attendances...)
				addAuditLocked(fmt.Sprintf("Пропустил: %s (%s)", b.ChildName, b.ParentName))
			case "canceled":
				if b.IsFreeTrial && fam != nil {
					fam.FreeTrialUsed = false
				}
				state.Attendances = append([]Attendance{{ID: state.NextID, FamilyID: b.FamilyID, DateFormatted: today, ChildName: b.ChildName, StatusLabel: "Отмена (место возвращено)"}}, state.Attendances...)
				addAuditLocked(fmt.Sprintf("Отмена записи: %s (%s)", b.ChildName, b.ParentName))
			}
			state.Bookings = append(state.Bookings[:idx], state.Bookings[idx+1:]...)
		}
	}
	saveStateLocked()
	mu.Unlock()
	http.Redirect(w, r, "/master", http.StatusSeeOther)
}


func freePort8080() {
	if runtime.GOOS == "windows" {
		out, err := exec.Command("cmd", "/C", "netstat -ano | findstr :8080 | findstr LISTENING").Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			myPID := strconv.Itoa(os.Getpid())
			for _, line := range lines {
				fields := strings.Fields(line)
				if len(fields) >= 5 {
					pid := fields[len(fields)-1]
					if pid != "" && pid != "0" && pid != myPID {
						_ = exec.Command("taskkill", "/F", "/PID", pid).Run()
					}
				}
			}
		}
	}
}

func main() {
	freePort8080()
	loadOrInitState()
	http.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/book", handleBook)
	http.HandleFunc("/parent-login", handleAuth)
	http.HandleFunc("/master-login", handleAuth)
	http.HandleFunc("/cabinet/", handleCabinet)
	http.HandleFunc("/master", handleMaster)
	http.HandleFunc("/master/logout", handleMasterPost)
	http.HandleFunc("/master/demo-reset", handleMasterPost)
	http.HandleFunc("/master/media/upload", handleMasterPost)
	http.HandleFunc("/master/slots/toggle", handleMasterPost)
	http.HandleFunc("/master/settings/schedule", handleMasterPost)
	http.HandleFunc("/master/settings/capacity", handleMasterPost)
	http.HandleFunc("/master/settings/sub-days", handleMasterPost)
	http.HandleFunc("/master/toggle-free-trial", handleMasterPost)
	http.HandleFunc("/master/payments/manual", handleMasterPost)
	http.HandleFunc("/master/payments/attach-receipt", handleMasterPost)
	http.HandleFunc("/master/attendance", handleMasterPost)

	http.HandleFunc("/master/families/", func(w http.ResponseWriter, r *http.Request) {
		if isMasterAuth(r) && r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/revoke-token") {
			trimmed := strings.TrimPrefix(r.URL.Path, "/master/families/")
			idStr, _, _ := strings.Cut(trimmed, "/")
			famID, _ := strconv.Atoi(idStr)
			mu.Lock()
			if fam := findFamilyLocked(famID); fam != nil {
				fam.Token = genToken()
				addAuditLocked(fmt.Sprintf("Перевыпущена ссылка для семьи %s", fam.ParentName))
				saveStateLocked()
			}
			mu.Unlock()
		}
		http.Redirect(w, r, "/master", http.StatusSeeOther)
	})

	http.HandleFunc("/master/export.csv", func(w http.ResponseWriter, r *http.Request) {
		if !isMasterAuth(r) {
			http.Redirect(w, r, "/master-login", http.StatusSeeOther)
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=yasna_clients.csv")
		mu.Lock()
		defer mu.Unlock()
		_, _ = fmt.Fprintln(w, "\uFEFFРодитель;Телефон;Email;Остаток_занятий;Ссылка_кабинета")
		for _, f := range state.Families {
			_, _ = fmt.Fprintf(w, "%s;%s;%s;%d;/cabinet/%s\n", f.ParentName, f.Phone, f.Email, f.LessonsRemaining, f.Token)
		}
	})

	http.HandleFunc("/privacy", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		logo := state.LogoURL
		mu.Unlock()
		renderPage(w, "legal.html", LegalData{BasePage: BasePage{PageTitle: "Политика конфиденциальности", LogoURL: logo}, DocTitle: "Политика обработки персональных данных (152-ФЗ)", DocText: "Столярная мастерская «Ясна» (г. Сургут, ул. 30 лет Победы, 21/3) обрабатывает имена, телефоны и Email исключительно для записи на занятия и информирования родителей."})
	})

	http.HandleFunc("/offer", func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		logo := state.LogoURL
		mu.Unlock()
		renderPage(w, "legal.html", LegalData{BasePage: BasePage{PageTitle: "Публичная оферта", LogoURL: logo}, DocTitle: "Договор публичной оферты на оказание услуг", DocText: "Настоящий документ определяет условия проведения занятий в детской столярной мастерской «Ясна» (г. Сургут)."})
	})

	http.HandleFunc("/robots.txt", func(w http.ResponseWriter, r *http.Request) { http.ServeFile(w, r, "robots.txt") })

	
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-stop
		mu.Lock()
		saveStateLocked()
		mu.Unlock()
		fmt.Println("\nБаза data.json сохранена. Сервер корректно остановлен.")
		os.Exit(0)
	}()

	fmt.Println("Единый портал мастерской «Ясна» запущен: http://localhost:8080/")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
