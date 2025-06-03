package user

import (
	"context"
	"log/slog"
	"os"
	"testing"

	"github.com/google/uuid"
)

// setupServiceTest initializes a new Service with mock dependencies for testing.
func setupServiceTest(t *testing.T) (*Service, *mockRepository, *mockPublisher) {
	repo := newMockRepository()
	pub := newMockPublisher()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil)) // Or use a testing logger if preferred
	service := NewService(repo, pub, logger)
	return service, repo, pub
}

type mockRepository struct {
	users map[uuid.UUID]*User
}

func newMockRepository() *mockRepository {
	return &mockRepository{
		users: make(map[uuid.UUID]*User),
	}
}

func (m *mockRepository) Create(ctx context.Context, u *User) error {
	if _, exists := m.users[u.ID]; exists {
		return ErrAlreadyExists
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockRepository) GetByID(ctx context.Context, id uuid.UUID) (*User, error) {
	if u, exists := m.users[id]; exists {
		return u, nil
	}
	return nil, ErrNotFound
}

func (m *mockRepository) GetByEmail(ctx context.Context, email string) (*User, error) {
	for _, u := range m.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepository) GetByNickname(ctx context.Context, nickname string) (*User, error) {
	for _, u := range m.users {
		if u.Nickname == nickname {
			return u, nil
		}
	}
	return nil, ErrNotFound
}

func (m *mockRepository) Update(ctx context.Context, u *User) error {
	if _, exists := m.users[u.ID]; !exists {
		return ErrNotFound
	}
	m.users[u.ID] = u
	return nil
}

func (m *mockRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if _, exists := m.users[id]; !exists {
		return ErrNotFound
	}
	delete(m.users, id)
	return nil
}

func (m *mockRepository) List(ctx context.Context, params ListParams) ([]User, int64, error) {
	var filteredUsers []User

	for _, u := range m.users {
		match := true
		for _, filter := range params.Filters {
			switch filter.Field {
			case "country":
				if u.Country != filter.Value {
					match = false
				}
			case "first_name":
				if u.FirstName != filter.Value {
					match = false
				}
			case "last_name":
				if u.LastName != filter.Value {
					match = false
				}
			case "nickname":
				if u.Nickname != filter.Value {
					match = false
				}
			case "email":
				if u.Email != filter.Value {
					match = false
				}
			}
		}
		if match {
			filteredUsers = append(filteredUsers, *u)
		}
	}

	totalCount := int64(len(filteredUsers))

	start := params.Offset
	if start > len(filteredUsers) {
		return []User{}, totalCount, nil
	}

	end := start + params.Limit
	if end > len(filteredUsers) {
		end = len(filteredUsers)
	}

	return filteredUsers[start:end], totalCount, nil
}

type mockPublisher struct {
	createdUsers []*User
	updatedUsers []*User
	deletedUsers []*User
}

func newMockPublisher() *mockPublisher {
	return &mockPublisher{
		createdUsers: make([]*User, 0),
		updatedUsers: make([]*User, 0),
		deletedUsers: make([]*User, 0),
	}
}

func (m *mockPublisher) PublishCreatedUser(ctx context.Context, user *User) error {
	m.createdUsers = append(m.createdUsers, user)
	return nil
}

func (m *mockPublisher) PublishUpdatedUser(ctx context.Context, user *User) error {
	m.updatedUsers = append(m.updatedUsers, user)
	return nil
}

func (m *mockPublisher) PublishDeletedUser(ctx context.Context, user *User) error {
	m.deletedUsers = append(m.deletedUsers, user)
	return nil
}

func TestService_CreateUser(t *testing.T) {
	service, _, pub := setupServiceTest(t)

	tests := []struct {
		name      string
		firstName string
		lastName  string
		nickname  string
		password  string
		email     string
		country   string
		wantErr   bool
	}{
		{
			name:      "valid user",
			firstName: "John",
			lastName:  "Doe",
			nickname:  "johndoe",
			password:  "secret123",
			email:     "john@example.com",
			country:   "US",
			wantErr:   false,
		},
		{
			name:      "duplicate email",
			firstName: "Jane",
			lastName:  "Doe",
			nickname:  "janedoe",
			password:  "secret123",
			email:     "john@example.com", // Same as above
			country:   "US",
			wantErr:   true,
		},
		{
			name:      "duplicate nickname",
			firstName: "Jane",
			lastName:  "Smith",
			nickname:  "johndoe", // Same as first test
			password:  "secret123",
			email:     "jane@example.com",
			country:   "US",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				FirstName: tt.firstName,
				LastName:  tt.lastName,
				Nickname:  tt.nickname,
				Password:  tt.password,
				Email:     tt.email,
				Country:   tt.country,
			}

			u, err := service.CreateUser(context.Background(), user)

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.CreateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(pub.createdUsers) == 0 {
					t.Error("Service.CreateUser() no event published")
				}

				lastUser := pub.createdUsers[len(pub.createdUsers)-1]
				if lastUser.ID != u.ID {
					t.Errorf("Service.CreateUser() published user ID = %v, want %v", lastUser.ID, u.ID)
				}
			}
		})
	}
}

func TestService_UpdateUser(t *testing.T) {
	service, _, pub := setupServiceTest(t)

	initialUser := &User{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Password:  "secret123",
		Email:     "john@example.com",
		Country:   "US",
	}
	user, err := service.CreateUser(context.Background(), initialUser)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name      string
		id        uuid.UUID
		firstName string
		lastName  string
		nickname  string
		email     string
		country   string
		wantErr   bool
	}{
		{
			name:      "valid update",
			id:        user.ID,
			firstName: "John",
			lastName:  "Smith",
			nickname:  "johnsmith",
			email:     "john.smith@example.com",
			country:   "UK",
			wantErr:   false,
		},
		{
			name:      "not found",
			id:        uuid.New(),
			firstName: "John",
			lastName:  "Smith",
			nickname:  "johnsmith",
			email:     "john.smith@example.com",
			country:   "UK",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updatedUser := &User{
				FirstName: tt.firstName,
				LastName:  tt.lastName,
				Nickname:  tt.nickname,
				Email:     tt.email,
				Country:   tt.country,
			}

			u, err := service.UpdateUser(context.Background(), tt.id, updatedUser)

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.UpdateUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(pub.updatedUsers) == 0 {
					t.Error("Service.UpdateUser() no event published")
				}

				lastUser := pub.updatedUsers[len(pub.updatedUsers)-1]
				if lastUser.ID != u.ID {
					t.Errorf("Service.UpdateUser() published user ID = %v, want %v", lastUser.ID, u.ID)
				}

				if u.FirstName != tt.firstName {
					t.Errorf("UpdateUser() FirstName = %v, want %v", u.FirstName, tt.firstName)
				}
				if u.LastName != tt.lastName {
					t.Errorf("UpdateUser() LastName = %v, want %v", u.LastName, tt.lastName)
				}
				if u.Nickname != tt.nickname {
					t.Errorf("UpdateUser() Nickname = %v, want %v", u.Nickname, tt.nickname)
				}
				if u.Email != tt.email {
					t.Errorf("UpdateUser() Email = %v, want %v", u.Email, tt.email)
				}
				if u.Country != tt.country {
					t.Errorf("UpdateUser() Country = %v, want %v", u.Country, tt.country)
				}
			}
		})
	}
}

func TestService_DeleteUser(t *testing.T) {
	service, _, pub := setupServiceTest(t)

	user := &User{
		FirstName: "John",
		LastName:  "Doe",
		Nickname:  "johndoe",
		Password:  "secret123",
		Email:     "john@example.com",
		Country:   "US",
	}
	createdUser, err := service.CreateUser(context.Background(), user)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "valid delete",
			id:      createdUser.ID,
			wantErr: false,
		},
		{
			name:    "not found",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.DeleteUser(context.Background(), tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.DeleteUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(pub.deletedUsers) == 0 {
					t.Error("Service.DeleteUser() no event published")
				}

				lastUser := pub.deletedUsers[len(pub.deletedUsers)-1]
				if lastUser.ID != tt.id {
					t.Errorf("Service.DeleteUser() published user ID = %v, want %v", lastUser.ID, tt.id)
				}

				_, err := service.GetUser(context.Background(), tt.id)
				if err == nil {
					t.Error("Service.DeleteUser() user still exists after deletion")
				}
			}
		})
	}
}

func TestService_ListUsers(t *testing.T) {
	service, _, _ := setupServiceTest(t)

	users := []*User{
		{
			FirstName: "John",
			LastName:  "Doe",
			Nickname:  "johndoe",
			Password:  "secret123",
			Email:     "john@example.com",
			Country:   "US",
		},
		{
			FirstName: "Jane",
			LastName:  "Smith",
			Nickname:  "janesmith",
			Password:  "secret123",
			Email:     "jane@example.com",
			Country:   "UK",
		},
		{
			FirstName: "Bob",
			LastName:  "Johnson",
			Nickname:  "bjohnson",
			Password:  "secret123",
			Email:     "bob@example.com",
			Country:   "US",
		},
	}

	for _, u := range users {
		_, err := service.CreateUser(context.Background(), u)
		if err != nil {
			t.Fatalf("Failed to create test user: %v", err)
		}
	}

	tests := []struct {
		name        string
		params      ListParams
		wantCount   int
		wantTotal   int64
		wantCountry string
		wantErr     bool
	}{
		{
			name: "list all users",
			params: ListParams{
				Limit:  10,
				Offset: 0,
			},
			wantCount: 3,
			wantTotal: 3,
			wantErr:   false,
		},
		{
			name: "filter by country",
			params: ListParams{
				Limit:  10,
				Offset: 0,
				Filters: []Filter{
					{Field: "country", Value: "US"},
				},
			},
			wantCount:   2,
			wantTotal:   2,
			wantCountry: "US",
			wantErr:     false,
		},
		{
			name: "pagination",
			params: ListParams{
				Limit:  1,
				Offset: 1,
			},
			wantCount: 1,
			wantTotal: 3,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			users, hasMore, total, err := service.ListUsers(context.Background(), tt.params)

			if (err != nil) != tt.wantErr {
				t.Errorf("Service.ListUsers() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if err == nil {
				if len(users) != tt.wantCount {
					t.Errorf("Service.ListUsers() got %v users, want %v", len(users), tt.wantCount)
				}

				if total != tt.wantTotal {
					t.Errorf("Service.ListUsers() total = %v, want %v", total, tt.wantTotal)
				}

				if tt.wantCountry != "" {
					for _, u := range users {
						if u.Country != tt.wantCountry {
							t.Errorf("Service.ListUsers() got user with country %v, want %v", u.Country, tt.wantCountry)
						}
					}
				}

				expectedHasMore := tt.params.Offset+tt.params.Limit < int(tt.wantTotal)
				if hasMore != expectedHasMore {
					t.Errorf("Service.ListUsers() hasMore = %v, want %v", hasMore, expectedHasMore)
				}
			}
		})
	}
}
