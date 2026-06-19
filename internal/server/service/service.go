package service

// Services aggregates the business services for wiring into transports.
type Services struct {
	User     *UserService
	Card     *CardService
	Password *PasswordService
	Note     *NoteService
	File     *FileService
}
