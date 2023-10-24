package app

import (
	_ "github.com/jackc/pgx/v5/stdlib"
)

// func TestLogin(t *testing.T) {
// 	gin.SetMode(gin.TestMode)
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	store := mocks.NewMockStore(ctrl)

// 	gomock.InOrder(
// 		store.EXPECT().GetUser(gomock.Any()).Return(
// 			&models.User{
// 				Login:    "a",
// 				Password: "$2a$07$me7lXx6x3fQpcrqxjYGa.eyFLQlwnZMI1kxCK8P90HCdUtol92936",
// 			}, nil),
// 		store.EXPECT().GetUser(gomock.Any()).Return(nil, originalStore.ErrLoginNotFound),
// 	)

// 	app := NewApp(config.GetDummy(), store, zap.L().Sugar())
// 	r, err := app.SetupRouter()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	srv := httptest.NewServer(r)
// 	defer srv.Close()

// 	tests := []struct {
// 		userCreds models.UserCredentialsSchema
// 		name      string
// 		url       string
// 		method    string
// 		status    int
// 	}{
// 		{
// 			name:      "Login user",
// 			userCreds: models.UserCredentialsSchema{Login: "a", Password: "b"},
// 			url:       "/api/user/login",
// 			status:    http.StatusOK,
// 			method:    http.MethodPost,
// 		},
// 		{
// 			name:      "Login not found",
// 			userCreds: models.UserCredentialsSchema{Login: "a", Password: "b"},
// 			url:       "/api/user/login",
// 			status:    http.StatusUnauthorized,
// 			method:    http.MethodPost,
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt

// 		b, err := json.Marshal(tt.userCreds)
// 		if err != nil {
// 			t.Error(err)
// 		}

// 		url, err := url.JoinPath(srv.URL, tt.url)
// 		if err != nil {
// 			t.Error(err)
// 		}

// 		req, err := http.NewRequest(tt.method, url, bytes.NewBuffer(b))
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if err := req.Body.Close(); err != nil {
// 			t.Error(err)
// 		}

// 		res, err := srv.Client().Do(req)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if err := res.Body.Close(); err != nil {
// 			t.Error(err)
// 		}
// 		require.Equal(t, tt.status, res.StatusCode)
// 	}
// }

// func TestRegister(t *testing.T) {
// 	gin.SetMode(gin.TestMode)
// 	ctrl := gomock.NewController(t)
// 	defer ctrl.Finish()

// 	store := mocks.NewMockStore(ctrl)

// 	gomock.InOrder(
// 		store.EXPECT().CreateUser(gomock.Any()).Return(int64(1), nil),
// 		store.EXPECT().CreateUser(gomock.Any()).Return(int64(0), originalStore.ErrDuplicateLogin),
// 	)

// 	app := NewApp(config.GetDummy(), store, zap.L().Sugar())
// 	r, err := app.SetupRouter()
// 	if err != nil {
// 		t.Error(err)
// 	}

// 	srv := httptest.NewServer(r)
// 	defer srv.Close()

// 	tests := []struct {
// 		userCreds models.UserCredentialsSchema
// 		name      string
// 		url       string
// 		method    string
// 		status    int
// 	}{
// 		{
// 			name:      "Register user",
// 			userCreds: models.UserCredentialsSchema{Login: "a", Password: "b"},
// 			url:       "/api/user/register",
// 			status:    http.StatusOK,
// 			method:    http.MethodPost,
// 		},
// 		{
// 			name:      "Register user with conflict",
// 			userCreds: models.UserCredentialsSchema{Login: "a", Password: "b"},
// 			url:       "/api/user/register",
// 			status:    http.StatusConflict,
// 			method:    http.MethodPost,
// 		},
// 	}

// 	for _, tt := range tests {
// 		tt := tt

// 		b, err := json.Marshal(tt.userCreds)
// 		if err != nil {
// 			t.Error(err)
// 		}

// 		url, err := url.JoinPath(srv.URL, tt.url)
// 		if err != nil {
// 			t.Error(err)
// 		}

// 		req, err := http.NewRequest(tt.method, url, bytes.NewBuffer(b))
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if err := req.Body.Close(); err != nil {
// 			t.Error(err)
// 		}

// 		res, err := srv.Client().Do(req)
// 		if err != nil {
// 			t.Error(err)
// 		}
// 		if err := res.Body.Close(); err != nil {
// 			t.Error(err)
// 		}
// 		require.Equal(t, tt.status, res.StatusCode)
// 		if tt.status == http.StatusOK {
// 			require.Contains(t, res.Header.Get("Set-Cookie"), "jwt")
// 		}
// 	}
// }
