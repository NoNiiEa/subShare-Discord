package service

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/NoNiiEa/subShare-Discord/src/database"
	"github.com/NoNiiEa/subShare-Discord/src/exception"
	"github.com/NoNiiEa/subShare-Discord/src/models"
	"github.com/NoNiiEa/subShare-Discord/src/okslip"
	"github.com/NoNiiEa/subShare-Discord/src/repository"

	_ "github.com/mattn/go-sqlite3"
)

const (
	testGuild   = "guild-1"
	testPayer   = "payer-1"
	testOwner   = "owner-1"
	testAccount = "0861234567"
)

// fakeOkSlip returns a canned response (or error) instead of calling the
// provider. The real client is an interface, so nothing else needs faking.
type fakeOkSlip struct {
	resp *okslip.Response
	err  error
	// calls counts provider hits: a batch must only ever verify the slip once.
	calls int
}

func (f *fakeOkSlip) CheckSlip(ctx context.Context, slipUrl string) (*okslip.Response, error) {
	f.calls++
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

// slipFor builds a fresh slip paid to testAccount by PromptPay.
func slipFor(amount float64, transRef string) *okslip.Response {
	return &okslip.Response{
		Success: true,
		Data: okslip.Data{
			Success:        true,
			TransRef:       transRef,
			TransTimestamp: time.Now().UTC(),
			Amount:         amount,
			Receiver: okslip.Party{
				Proxy: okslip.Proxy{Type: "MSISDN", Value: "086xxx4567"},
			},
		},
	}
}

type fixture struct {
	svc       BillService
	db        *sql.DB
	billRepo  repository.BillRepository
	groupRepo repository.GroupRepository
	slip      *fakeOkSlip
}

func newFixture(t *testing.T, slip *fakeOkSlip) *fixture {
	t.Helper()

	// A distinct in-memory database per test; the service calls BeginTx directly,
	// so a real DB is required rather than a fake repository.
	dsn := fmt.Sprintf("file:%s?mode=memory&cache=shared", t.Name())
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })

	if err := database.CreateTable(db); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	billRepo := repository.NewBillRepository(db)
	groupRepo := repository.NewGroupRepository(db)
	slipRepo := repository.NewSlipRepository(db)

	return &fixture{
		svc:       NewBillService(db, billRepo, groupRepo, slipRepo, slip),
		db:        db,
		billRepo:  billRepo,
		groupRepo: groupRepo,
		slip:      slip,
	}
}

// seedGroup creates a group with the payer as a member carrying the given debt.
func (f *fixture) seedGroup(t *testing.T, name string, dept int, owner string, account string) *models.Group {
	t.Helper()

	g, err := f.groupRepo.Create(context.Background(), &models.Group{
		Name:            name,
		Amount:          dept,
		AmountPerMember: dept,
		DueDay:          5,
		DiscordGuildID:  testGuild,
		OwnerDiscordID:  owner,
		Payment:         models.PaymentAccount{Method: models.PromptPay, Account: account},
		Members: []models.GroupMember{
			{MemberID: owner, Dept: 0, Status: models.MemberStatusActive, Payment: models.PaymentStatusPaid},
			{MemberID: testPayer, Dept: dept, Status: models.MemberStatusActive, Payment: models.PaymentStatusNotPaid},
		},
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		t.Fatalf("seed group: %v", err)
	}
	return g
}

func (f *fixture) seedBill(t *testing.T, groupId int, amount int, member string) *models.Bill {
	t.Helper()

	now := time.Now().UTC()
	b := &models.Bill{
		GroupID:   groupId,
		GuildID:   testGuild,
		MemberID:  member,
		Year:      2026,
		Month:     7,
		AmountDue: amount,
		Currency:  defaultCurrency,
		Status:    models.BillStatusPending,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := f.billRepo.Create(context.Background(), b); err != nil {
		t.Fatalf("seed bill: %v", err)
	}

	// Create does not populate the generated id; find it by re-reading the group's bills.
	bills, err := f.billRepo.GetByGroupID(context.Background(), groupId)
	if err != nil {
		t.Fatalf("read back bill: %v", err)
	}
	return &bills[len(bills)-1]
}

func (f *fixture) memberDept(t *testing.T, groupId int, member string) int {
	t.Helper()

	g, err := f.groupRepo.GetById(context.Background(), groupId)
	if err != nil {
		t.Fatalf("get group: %v", err)
	}
	for _, m := range g.Members {
		if m.MemberID == member {
			return m.Dept
		}
	}
	t.Fatalf("member %s not in group %d", member, groupId)
	return 0
}

func (f *fixture) billStatus(t *testing.T, billId int) models.BillStatus {
	t.Helper()

	b, err := f.billRepo.GetById(context.Background(), billId)
	if err != nil {
		t.Fatalf("get bill %d: %v", billId, err)
	}
	return b.Status
}

func (f *fixture) countBills(t *testing.T) int {
	t.Helper()

	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM bills`).Scan(&n); err != nil {
		t.Fatalf("count bills: %v", err)
	}
	return n
}

func (f *fixture) countSlips(t *testing.T) int {
	t.Helper()

	var n int
	if err := f.db.QueryRow(`SELECT count(*) FROM slips`).Scan(&n); err != nil {
		t.Fatalf("count slips: %v", err)
	}
	return n
}

func TestPayMultiple_AcrossTwoGroups(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(250, "ref-1")})

	g1 := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
	g2 := f.seedGroup(t, "Spotify", 100, testOwner, testAccount)
	b1 := f.seedBill(t, g1.ID, 150, testPayer)
	b2 := f.seedBill(t, g2.ID, 100, testPayer)

	res, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b1.ID, b2.ID}, "http://slip")
	if err != nil {
		t.Fatalf("PayMultiple: %v", err)
	}

	if res.TotalDue != 250 || res.AmountPaid != 250 || len(res.Bills) != 2 {
		t.Errorf("got total=%d paid=%v bills=%d, want 250/250/2", res.TotalDue, res.AmountPaid, len(res.Bills))
	}
	if f.slip.calls != 1 {
		t.Errorf("provider called %d times, want exactly 1 for the batch", f.slip.calls)
	}
	for _, b := range []*models.Bill{b1, b2} {
		if got := f.billStatus(t, b.ID); got != models.BillStatusVerified {
			t.Errorf("bill %d status = %q, want verified", b.ID, got)
		}
	}
	if got := f.memberDept(t, g1.ID, testPayer); got != 0 {
		t.Errorf("group %d dept = %d, want 0", g1.ID, got)
	}
	if got := f.memberDept(t, g2.ID, testPayer); got != 0 {
		t.Errorf("group %d dept = %d, want 0", g2.ID, got)
	}
	if got := f.countSlips(t); got != 1 {
		t.Errorf("slips = %d, want 1", got)
	}
	// No "balance remaining" bill: the slip covered the total.
	if got := f.countBills(t); got != 2 {
		t.Errorf("bills = %d, want 2 (no remainder bill)", got)
	}
}

// Two bills in one group must each reduce the debt. Reloading the group per bill
// would make the second write clobber the first.
func TestPayMultiple_TwoBillsSameGroup(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(300, "ref-2")})

	g := f.seedGroup(t, "Netflix", 300, testOwner, testAccount)
	b1 := f.seedBill(t, g.ID, 150, testPayer)
	b2 := f.seedBill(t, g.ID, 150, testPayer)

	if _, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b1.ID, b2.ID}, "http://slip"); err != nil {
		t.Fatalf("PayMultiple: %v", err)
	}

	if got := f.memberDept(t, g.ID, testPayer); got != 0 {
		t.Errorf("dept = %d, want 0 — both bills must be credited to the same group instance", got)
	}
}

// A single-group overpayment credits the surplus against the remaining debt,
// rather than losing it.
func TestPayMultiple_OverpaymentCreditsSurplusToGroup(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(500, "ref-3")})

	// Debt is 400 but only a 150 bill is being paid, with a 500 slip.
	g := f.seedGroup(t, "Netflix", 400, testOwner, testAccount)
	b := f.seedBill(t, g.ID, 150, testPayer)

	res, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b.ID}, "http://slip")
	if err != nil {
		t.Fatalf("PayMultiple: %v", err)
	}

	if res.AmountPaid != 500 || res.TotalDue != 150 {
		t.Errorf("got paid=%v total=%d, want 500/150", res.AmountPaid, res.TotalDue)
	}
	if res.SurplusCredited != 350 {
		t.Errorf("surplus credited = %v, want 350", res.SurplusCredited)
	}
	// 400 debt - 150 bill - 350 surplus, clamped at zero.
	if got := f.memberDept(t, g.ID, testPayer); got != 0 {
		t.Errorf("dept = %d, want 0 — the surplus must reduce the debt", got)
	}
	if got := f.countBills(t); got != 1 {
		t.Errorf("bills = %d, want 1 (surplus must not spawn a bill)", got)
	}
}

// Across several groups there is no principled way to split a surplus, so it is
// left uncredited rather than being applied arbitrarily.
func TestPayMultiple_OverpaymentAcrossGroupsIsNotCredited(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(500, "ref-3b")})

	g1 := f.seedGroup(t, "Netflix", 200, testOwner, testAccount)
	g2 := f.seedGroup(t, "Spotify", 200, testOwner, testAccount)
	b1 := f.seedBill(t, g1.ID, 150, testPayer)
	b2 := f.seedBill(t, g2.ID, 100, testPayer)

	res, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b1.ID, b2.ID}, "http://slip")
	if err != nil {
		t.Fatalf("PayMultiple: %v", err)
	}

	if res.SurplusCredited != 0 {
		t.Errorf("surplus credited = %v, want 0 across multiple groups", res.SurplusCredited)
	}
	if got := f.memberDept(t, g1.ID, testPayer); got != 50 {
		t.Errorf("group %d dept = %d, want 50 (200 - 150 bill only)", g1.ID, got)
	}
	if got := f.memberDept(t, g2.ID, testPayer); got != 100 {
		t.Errorf("group %d dept = %d, want 100 (200 - 100 bill only)", g2.ID, got)
	}
}

func TestPayMultiple_DuplicateIdsCollapse(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(150, "ref-4")})

	g := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
	b := f.seedBill(t, g.ID, 150, testPayer)

	res, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b.ID, b.ID, b.ID}, "http://slip")
	if err != nil {
		t.Fatalf("PayMultiple: %v", err)
	}

	if res.TotalDue != 150 || len(res.Bills) != 1 {
		t.Errorf("got total=%d bills=%d, want 150/1 — the same bill must be counted once", res.TotalDue, len(res.Bills))
	}
}

// A rejected batch must leave the database exactly as it was.
func TestPayMultiple_InsufficientSlipChangesNothing(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(200, "ref-5")})

	g1 := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
	g2 := f.seedGroup(t, "Spotify", 100, testOwner, testAccount)
	b1 := f.seedBill(t, g1.ID, 150, testPayer)
	b2 := f.seedBill(t, g2.ID, 100, testPayer)

	_, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b1.ID, b2.ID}, "http://slip")
	if !errors.Is(err, exception.ErrSlipInsufficient) {
		t.Fatalf("err = %v, want ErrSlipInsufficient", err)
	}

	if got := f.billStatus(t, b1.ID); got != models.BillStatusPending {
		t.Errorf("bill %d status = %q, want pending", b1.ID, got)
	}
	if got := f.billStatus(t, b2.ID); got != models.BillStatusPending {
		t.Errorf("bill %d status = %q, want pending", b2.ID, got)
	}
	if got := f.memberDept(t, g1.ID, testPayer); got != 150 {
		t.Errorf("group %d dept = %d, want 150 (unchanged)", g1.ID, got)
	}
	if got := f.countSlips(t); got != 0 {
		t.Errorf("slips = %d, want 0 — a rejected batch must not record a slip", got)
	}
}

func TestPayMultiple_RejectsMixedPayee(t *testing.T) {
	otherAccount := "0899999999"

	tests := []struct {
		name        string
		owner2      string
		account2    string
		wantErr     error
		description string
	}{
		{"different account, same owner", testOwner, otherAccount, exception.ErrMixedPayee, "one slip has one receiver"},
		{"different owner, same account", "owner-2", testAccount, exception.ErrMixedPayee, "owner must match too"},
		{"same owner and account", testOwner, testAccount, nil, "batchable"},
		{"same account, different formatting", testOwner, "086-123-4567", nil, "digits-only comparison"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, &fakeOkSlip{resp: slipFor(250, "ref-"+tt.name)})

			g1 := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
			g2 := f.seedGroup(t, "Spotify", 100, tt.owner2, tt.account2)
			b1 := f.seedBill(t, g1.ID, 150, testPayer)
			b2 := f.seedBill(t, g2.ID, 100, testPayer)

			_, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b1.ID, b2.ID}, "http://slip")
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v (%s)", err, tt.wantErr, tt.description)
			}
		})
	}
}

func TestPayMultiple_Rejections(t *testing.T) {
	tests := []struct {
		name    string
		billIds func(ownBill, foreignBill, paidBill int) []int
		wantErr error
	}{
		{"empty selection", func(_, _, _ int) []int { return nil }, exception.ErrNoBillsSelected},
		{"unknown bill id", func(own, _, _ int) []int { return []int{own, 9999} }, exception.ErrBillNotFound},
		{"someone else's bill", func(own, foreign, _ int) []int { return []int{own, foreign} }, exception.ErrNotMatch},
		{"already paid bill", func(own, _, paid int) []int { return []int{own, paid} }, exception.ErrBillAlreadyPaid},
		{"invalid id", func(own, _, _ int) []int { return []int{own, 0} }, exception.ErrInvalidID},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newFixture(t, &fakeOkSlip{resp: slipFor(1000, "ref-"+tt.name)})

			g := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
			own := f.seedBill(t, g.ID, 150, testPayer)
			foreign := f.seedBill(t, g.ID, 150, "someone-else")

			paid := f.seedBill(t, g.ID, 150, testPayer)
			paid.Status = models.BillStatusVerified
			if err := f.billRepo.Update(context.Background(), paid.ID, paid); err != nil {
				t.Fatalf("mark paid: %v", err)
			}

			_, err := f.svc.PayMultiple(
				context.Background(), testPayer, testGuild,
				tt.billIds(own.ID, foreign.ID, paid.ID), "http://slip",
			)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}

			// Every rejection happens before the provider call, so it costs nothing.
			if f.slip.calls != 0 {
				t.Errorf("provider called %d times, want 0 — reject before paying for verification", f.slip.calls)
			}
		})
	}
}

func TestPayMultiple_TooManyBills(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(1, "ref-many")})

	ids := make([]int, maxBatchBills+1)
	for i := range ids {
		ids[i] = i + 1
	}

	_, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, ids, "http://slip")
	if !errors.Is(err, exception.ErrTooManyBills) {
		t.Fatalf("err = %v, want ErrTooManyBills", err)
	}
}

func TestPayMultiple_RejectsStaleSlip(t *testing.T) {
	stale := slipFor(150, "ref-stale")
	stale.Data.TransTimestamp = time.Now().UTC().Add(-2 * slipFreshnessWindow)

	f := newFixture(t, &fakeOkSlip{resp: stale})
	g := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
	b := f.seedBill(t, g.ID, 150, testPayer)

	_, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b.ID}, "http://slip")
	if !errors.Is(err, exception.ErrSlipExpired) {
		t.Fatalf("err = %v, want ErrSlipExpired", err)
	}
}

func TestPayMultiple_RejectsReusedSlip(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(150, "ref-reused")})

	g := f.seedGroup(t, "Netflix", 300, testOwner, testAccount)
	b1 := f.seedBill(t, g.ID, 150, testPayer)
	b2 := f.seedBill(t, g.ID, 150, testPayer)

	if _, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b1.ID}, "http://slip"); err != nil {
		t.Fatalf("first payment: %v", err)
	}

	// Same slip (same transRef) submitted again for a different bill.
	_, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b2.ID}, "http://slip")
	if !errors.Is(err, exception.ErrDupeSlip) {
		t.Fatalf("err = %v, want ErrDupeSlip", err)
	}
	if got := f.billStatus(t, b2.ID); got != models.BillStatusPending {
		t.Errorf("bill %d status = %q, want pending", b2.ID, got)
	}
}

func TestPayMultiple_RejectsWrongReceiver(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(150, "ref-wrong")})

	// Group expects a different account than the slip was paid to.
	g := f.seedGroup(t, "Netflix", 150, testOwner, "0812223333")
	b := f.seedBill(t, g.ID, 150, testPayer)

	_, err := f.svc.PayMultiple(context.Background(), testPayer, testGuild, []int{b.ID}, "http://slip")
	if !errors.Is(err, exception.ErrWrongReciever) {
		t.Fatalf("err = %v, want ErrWrongReciever", err)
	}
}

// Pay must keep its lenient underpayment behaviour: the bill is still verified,
// the full slip amount is credited, and a "balance remaining" bill is created.
func TestPay_UnderpaymentStillCreatesRemainderBill(t *testing.T) {
	f := newFixture(t, &fakeOkSlip{resp: slipFor(100, "ref-under")})

	g := f.seedGroup(t, "Netflix", 150, testOwner, testAccount)
	b := f.seedBill(t, g.ID, 150, testPayer)

	paid, err := f.svc.Pay(context.Background(), testPayer, testGuild, b.ID, "http://slip")
	if err != nil {
		t.Fatalf("Pay: %v", err)
	}

	if paid.Status != models.BillStatusVerified {
		t.Errorf("status = %q, want verified", paid.Status)
	}
	if got := f.memberDept(t, g.ID, testPayer); got != 50 {
		t.Errorf("dept = %d, want 50 (150 - 100 paid)", got)
	}
	if got := f.countBills(t); got != 2 {
		t.Fatalf("bills = %d, want 2 (original + balance remaining)", got)
	}

	bills, err := f.billRepo.GetByGroupID(context.Background(), g.ID)
	if err != nil {
		t.Fatalf("read bills: %v", err)
	}
	remainder := bills[len(bills)-1]
	if remainder.AmountDue != 50 {
		t.Errorf("remainder amount = %d, want 50", remainder.AmountDue)
	}
	if remainder.Status != models.BillStatusPending {
		t.Errorf("remainder status = %q, want pending", remainder.Status)
	}
}
