package service

// Services aggregates the business services for wiring into transports.
type Services struct {
	User     *UserService
	Card     *EntryService
	Password *EntryService
	Note     *EntryService
	File     *FileService
}
