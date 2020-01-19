package lease

// Writer put an item into the data store
type Writer interface {
	WriteLease(input *Lease, lastModifiedOn *int64) error
}

// Deleter Deletes an item from the data store
type Deleter interface {
	DeleteLease(input *Lease) error
}

// SingleReader Reads an item information from the data store
type SingleReader interface {
	GetLeaseByID(leaseID string) (*Lease, error)
}

// MultipleReader reads multiple items from the data store
type MultipleReader interface {
	GetLeases(*Lease) (*Leases, error)
}

// Reader data Layer
type Reader interface {
	SingleReader
	MultipleReader
}

// WriterDeleter data layer
type WriterDeleter interface {
	Writer
	Deleter
}

// ReaderWriterDeleter includes Reader and Writer interfaces
type ReaderWriterDeleter interface {
	Reader
	WriterDeleter
}

// Eventer for publishing events
type Eventer interface {
	Publish() error
}

// Manager manages all the actions against a lease
type Manager interface {
	Setup(adminRole string) error
}

// Service is a type corresponding to a Account table record
type Service struct {
	dataSvc ReaderWriterDeleter
}

// NewServiceInput Input for creating a new Service
type NewServiceInput struct {
	DataSvc ReaderWriterDeleter
}

// NewService creates a new instance of the Service
func NewService(input NewServiceInput) *Service {
	return &Service{
		dataSvc: input.DataSvc,
	}
}
