package db

type User struct {
	UserName string `json:"user_name" gorm:"primaryKey"`
	UserID   string `json:"user_id"`
	IsOnline bool   `json:"is_online"`
}

//Create use Create
func (m *User) Create() error {
	return db.Create(m).Error
}

func (m *User) Delete() error {
	return db.Delete(m).Error
}


func (m *User) UpdateState(isOnline bool) error {
	return db.Model(m).
		Where("user_name=?", m.UserName).
		Update("is_online", isOnline).
		Error
}

func (m *User) Update() error {
	return db.Updates(m).Error
}

func GetUserByName(userName string) (user *User, err error) {
	user = new(User)
	err = db.Model(&User{}).
		Where("user_name=?", userName).
		First(user).Error

	return
}

func GetUserList() (users []User, err error) {

	users = make([]User, 0)

	err = db.Find(&users).Error

	return
}
