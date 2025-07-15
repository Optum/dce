package main

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"

	"github.com/Optum/dce/pkg/db"
	"github.com/stretchr/testify/assert"
)

// mockDB implements db.DBer for testing
type mockDB struct {
    accounts []db.Account
    err      error
}

func (m *mockDB) ScanAccounts(filter map[string]interface{}) ([]db.Account, error) {
    return m.accounts, m.err
}

func (m *mockDB) FindAccountsByStatus(status db.AccountStatus) ([]*db.Account, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    var filtered []*db.Account
    for i := range m.accounts {
        if m.accounts[i].AccountStatus == status {
            accountCopy := m.accounts[i]
            filtered = append(filtered, &accountCopy)
        }
    }
    
    return filtered, nil
}

func (m *mockDB) FindLeasesByAccount(accountID string) ([]*db.Lease, error) {
    // This is a stub implementation since it's not used in these tests
    return []*db.Lease{}, m.err
}

func (m *mockDB) FindLeasesByPrincipal(principalID string) ([]*db.Lease, error) {
    // This is a stub implementation since it's not used in these tests
    return []*db.Lease{}, m.err
}

func (m *mockDB) FindLeasesByStatus(status db.LeaseStatus) ([]*db.Lease, error) {
    // This is a stub implementation since it's not used in these tests
    return []*db.Lease{}, m.err
}

func (m *mockDB) GetAccount(accountID string) (*db.Account, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    for i := range m.accounts {
        if m.accounts[i].ID == accountID {
            accountCopy := m.accounts[i]
            return &accountCopy, nil
        }
    }
    
    return nil, errors.New("account not found")
}

func (m *mockDB) GetLease(leaseID string, accountID string) (*db.Lease, error) {
    // This is a stub implementation since it's not used in these tests
    return nil, m.err
}

// GetLeases is a stub implementation to satisfy db.DBer interface
func (m *mockDB) GetLeases(input db.GetLeasesInput) (db.GetLeasesOutput, error) {
    // This is a stub implementation since it's not used in these tests
    return db.GetLeasesOutput{
        Leases: []*db.Lease{},
    }, m.err
}

// GetLeaseByID is a stub implementation to satisfy db.DBer interface
func (m *mockDB) GetLeaseByID(leaseID string) (*db.Lease, error) {
    // This is a stub implementation since it's not used in these tests
    return nil, m.err
}

// GetReadyAccount is a stub implementation to satisfy db.DBer interface
func (m *mockDB) GetReadyAccount() (*db.Account, error) {
    if m.err != nil {
        return nil, m.err
    }
    for i := range m.accounts {
        if m.accounts[i].AccountStatus == db.Ready {
            accountCopy := m.accounts[i]
            return &accountCopy, nil
        }
    }
    return nil, errors.New("ready account not found")
}

// OrphanAccount is a stub implementation to satisfy db.DBer interface
func (m *mockDB) OrphanAccount(accountID string) (*db.Account, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    // Find and return the account being orphaned
    for i := range m.accounts {
        if m.accounts[i].ID == accountID {
            accountCopy := m.accounts[i]
            return &accountCopy, nil
        }
    }
    
    // Return nil account if not found (or you could return an error)
    return nil, errors.New("account not found")
}

func TestListNotReadyAccountsToCSV_Success(t *testing.T) {
    tmpfile, err := ioutil.TempFile("", "not_ready_accounts_*.csv")
    assert.NoError(t, err)
    defer os.Remove(tmpfile.Name())

    mockAccounts := []db.Account{
        {ID: "123", AccountStatus: db.NotReady, LastModifiedOn: 1719493200}, // 2024-06-27T12:00:00Z
        {ID: "456", AccountStatus: db.NotReady, LastModifiedOn: 1719496800}, // 2024-06-27T13:00:00Z
    }
    dbSvc := &mockDB{accounts: mockAccounts}

    // Use a dummy bucket and key since we are not actually uploading in this test
    err = listNotReadyAccountsToCSV(dbSvc, tmpfile.Name(), "dummy-bucket", "dummy-key")
    assert.NoError(t, err)

    // Check that the CSV file was written
    content, err := ioutil.ReadFile(tmpfile.Name())
    assert.NoError(t, err)
    assert.Contains(t, string(content), "AccountID")
    assert.Contains(t, string(content), "123")
    assert.Contains(t, string(content), "456")
}

func TestListNotReadyAccountsToCSV_DBError(t *testing.T) {
    dbSvc := &mockDB{err: errors.New("db error")}
    err := listNotReadyAccountsToCSV(dbSvc, "dummy.csv", "dummy-bucket", "dummy-key")
    assert.Error(t, err)
}

func TestListNotReadyAccountsToCSV_CSVWriteError(t *testing.T) {
    // Simulate a file that cannot be created
    dbSvc := &mockDB{accounts: []db.Account{{ID: "123", AccountStatus: db.NotReady, LastModifiedOn: 1719493200}}}
    // Use an invalid file path to force an error
    err := listNotReadyAccountsToCSV(dbSvc, "/invalid/path/not_ready_accounts.csv", "dummy-bucket", "dummy-key")
    assert.Error(t, err)
}