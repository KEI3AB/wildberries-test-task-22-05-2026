package domain

// Payload для кафки
type SearchEvent struct {
	SessionID       string
	Timestamp       int64  // миллисекунды
	NormalizedQuery string // текст запроса пользователя, сервис поиска по идее его должен был сделать нормальным
}
