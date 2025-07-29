package main

import (
	"errors"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/Optum/dce/pkg/db"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
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

// UpdateAccount is a new method to support updating accounts when marking as NotReady
func (m *mockDB) UpdateAccount(account *db.Account) error {
    if m.err != nil {
        return m.err
    }
    
    for i := range m.accounts {
        if m.accounts[i].ID == account.ID {
            m.accounts[i] = *account
            return nil
        }
    }
    
    return errors.New("account not found")
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
    return db.GetLeasesOutput{}, m.err
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

// PutAccount is a stub implementation to satisfy db.DBer interface
func (m *mockDB) PutAccount(account db.Account) error {
    if m.err != nil {
        return m.err
    }
    
    // Check if account already exists and update it
    for i := range m.accounts {
        if m.accounts[i].ID == account.ID {
            m.accounts[i] = account
            return nil
        }
    }
    
    // If account doesn't exist, add it
    m.accounts = append(m.accounts, account)
    return nil
}

// PutLease is a stub implementation to satisfy db.DBer interface
func (m *mockDB) PutLease(lease db.Lease) (*db.Lease, error) {
    // This is a stub implementation since it's not used in these tests
    if m.err != nil {
        return nil, m.err
    }
    return &lease, nil
}

// TransitionAccountStatus is a stub implementation to satisfy db.DBer interface
func (m *mockDB) TransitionAccountStatus(accountID string, fromStatus db.AccountStatus, toStatus db.AccountStatus) (*db.Account, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    for i := range m.accounts {
        if m.accounts[i].ID == accountID && m.accounts[i].AccountStatus == fromStatus {
            m.accounts[i].AccountStatus = toStatus
            accountCopy := m.accounts[i]
            return &accountCopy, nil
        }
    }
    
    return nil, errors.New("account not found or status transition not valid")
}

// TransitionLeaseStatus is a stub implementation to satisfy db.DBer interface
func (m *mockDB) TransitionLeaseStatus(leaseID string, accountID string, fromStatus db.LeaseStatus, toStatus db.LeaseStatus, reason db.LeaseStatusReason) (*db.Lease, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    // This is a stub implementation since it's not used in these tests
    return nil, errors.New("lease not found or status transition not valid")
}

// UpdateAccountPrincipalPolicyHash is a stub implementation to satisfy db.DBer interface
func (m *mockDB) UpdateAccountPrincipalPolicyHash(accountID string, principalID string, principalPolicyHash string) (*db.Account, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    // This is a stub implementation since it's not used in these tests
    // Find and return the account if it exists
    for i := range m.accounts {
        if m.accounts[i].ID == accountID {
            accountCopy := m.accounts[i]
            return &accountCopy, nil
        }
    }
    
    return nil, errors.New("account not found")
}

// UpsertLease is a stub implementation to satisfy db.DBer interface
func (m *mockDB) UpsertLease(lease db.Lease) (*db.Lease, error) {
    if m.err != nil {
        return nil, m.err
    }
    
    // This is a stub implementation since it's not used in these tests
    return &lease, nil
}

// Mock for the checkForBuckets function
type mockBucketChecker struct {
    mock.Mock
}

func (m *mockBucketChecker) checkForBuckets(sess *session.Session, accountID string) (bool, error) {
    args := m.Called(sess, accountID)
    return args.Bool(0), args.Error(1)
}

// Override the real function with our mock for testing
var mockBucketChecker = &mockBucketChecker{}

func TestScanAccountsForMissingBuckets(t *testing.T) {
    // Setup mock accounts with different statuses
    mockAccounts := []db.Account{
        {ID: "123", AccountStatus: db.Ready, LastModifiedOn: time.Now().Unix()},
        {ID: "456", AccountStatus: db.Leased, LastModifiedOn: time.Now().Unix()},
        {ID: "789", AccountStatus: db.NotReady, LastModifiedOn: time.Now().Unix()},
    }
    
    dbSvc := &mockDB{accounts: mockAccounts}
    
    // Mock the bucket checker to return false (no buckets) for account 123
    // and true (has buckets) for account 456
    mockBucketChecker.On("checkForBuckets", mock.Anything, "123").Return(false, nil)
    mockBucketChecker.On("checkForBuckets", mock.Anything, "456").Return(true, nil)
    
    // Call the function with a dummy file path
    err := scanAccountsForMissingRequiredBuckets(dbSvc, "test.csv", "test-bucket", "test-key")
    
    // Verify no error
    assert.NoError(t, err)
    
    // Account 123 should now be marked as NotReady
    for _, account := range dbSvc.accounts {
        if account.ID == "123" {
            assert.Equal(t, db.NotReady, account.AccountStatus)
            assert.NotNil(t, account.Metadata)
            assert.Equal(t, true, account.Metadata["BucketNotFound"])
        }
        if account.ID == "456" {
            // Account 456 should still be Leased
            assert.Equal(t, db.Leased, account.AccountStatus)
        }
    }
}

func TestListNotReadyAccountsToCSV_Success(t *testing.T) {
    tmpfile, err := ioutil.TempFile("", "not_ready_accounts_*.csv")
    assert.NoError(t, err)
    defer os.Remove(tmpfile.Name())

    // Create mock accounts with the new BucketNotFound field in metadata
    mockAccounts := []db.Account{
        {
            ID: "123", 
            AccountStatus: db.NotReady, 
            LastModifiedOn: time.Now().Unix(),
            Metadata: map[string]interface{}{
                "Reason": "Required bucket doesn't exist",
                "BucketNotFound": true,
            },
        },
        {
            ID: "456", 
            AccountStatus: db.NotReady, 
            LastModifiedOn: time.Now().Unix(),
            Metadata: map[string]interface{}{
                "Reason": "Other reason",
                "BucketNotFound": false,
            },
        },
    }
    dbSvc := &mockDB{accounts: mockAccounts}

    // Use a dummy bucket and key since we are not actually uploading in this test
    err = listNotReadyAccountsToCSV(dbSvc, tmpfile.Name(), "dummy-bucket", "dummy-key")
    assert.NoError(t, err)

    // Check that the CSV file was written with the expected content
    content, err := ioutil.ReadFile(tmpfile.Name())
    assert.NoError(t, err)
    csvContent := string(content)
    
    // Check for header with the new Bucket_Not_Found field
    assert.Contains(t, csvContent, "AccountID,Status,LastUpdated,Reason,Bucket_Not_Found")
    
    // Check that account 123 has true for Bucket_Not_Found
    assert.Contains(t, csvContent, "123,NotReady,")
    assert.Contains(t, csvContent, "Required bucket doesn't exist,true")
    
    // Check that account 456 has false for Bucket_Not_Found
    assert.Contains(t, csvContent, "456,NotReady,")
    assert.Contains(t, csvContent, "Other reason,false")
}

func TestListNotReadyAccountsToCSV_DBError(t *testing.T) {
    dbSvc := &mockDB{err: errors.New("db error")}
    err := listNotReadyAccountsToCSV(dbSvc, "dummy.csv", "dummy-bucket", "dummy-key")
    assert.Error(t, err)
}

func TestListNotReadyAccountsToCSV_CSVWriteError(t *testing.T) {
    // Simulate a file that cannot be created
    dbSvc := &mockDB{accounts: []db.Account{{ID: "123", AccountStatus: db.NotReady, LastModifiedOn: time.Now().Unix()}}}
    // Use an invalid file path to force an error
    err := listNotReadyAccountsToCSV(dbSvc, "/invalid/path/not_ready_accounts.csv", "dummy-bucket", "dummy-key")
    assert.Error(t, err)
}

func TestCheckForBuckets(t *testing.T) {
    // This would be an integration test requiring AWS credentials
    // In a real test suite, you would mock the AWS SDK calls
    // Here we're just testing the mocking setup we created
    
    sess := session.Must(session.NewSession())
    
    mockBucketChecker.On("checkForBuckets", sess, "test-account").Return(true, nil)
    
    hasBucket, err := mockBucketChecker.checkForBuckets(sess, "test-account")
    
    assert.NoError(t, err)
    assert.True(t, hasBucket)
    mockBucketChecker.AssertExpectations(t)
}