package valid

import (
	"log"
	"os"
	"testing"
)

type TestDeepStruct struct {
	Name  string `json:"name" validate:"required|string"`
	Phone string `json:"phone" validate:"phone_with_code"`
	Email string `json:"email" validate:"required|email"`
}
type TestStruct struct {
	Name        string            `json:"name" validate:"required>Name is required|string|from:1,5"`
	Description string            `json:"description" validate:"required|ascii|max:255"`
	Phone       string            `json:"phone" validate:"required|phone"`
	Username    string            `json:"username" validate:"required|username"`
	Terms       bool              `json:"terms" validate:"required"`
	Items       []string          `json:"items" validate:"required|slice:max:2|min:6|max:6"`
	Contact     *TestDeepStruct   `json:"contact" validate:"_"`
	Contacts    []*TestDeepStruct `json:"contacts" validate:"required|min:1"`
	Allergy     []int             `json:"allergy" validate:"int"`
}

var myLogger = log.New(os.Stdout, "Message:\t", log.Ldate|log.Ltime|log.Lshortfile)

func TestMain(m *testing.M) {
	request := &TestStruct{
		Name:        "Seyram",
		Description: "Nihil quis dolorum temporibus vitae delectus dolorem. Quasi debitis rem repudiandae odit repudiandae quae debitis amet recusandae. Qui repudiandae laudantium dicta voluptatibus ea quibusdam voluptatum reprehenderit recusandae. At laudantium sed repellat maiores adipisci rerum ex. Nam aliquam iste esse voluptates nemo. Libero incidunt alias amet odio quae.",
		Phone:       "943.406.7611",
		Username:    "admin@foodivoire.com",
		Terms:       false,
		Items:       []string{"quis Ut", "nulla amet cupidatat", "consectetur in", "veniam", "laboris incididunt ut culpa"},
		Contact:     &TestDeepStruct{Name: "Wood White", Phone: "++233265518694", Email: "contact@mail.com"},
		Contacts:    []*TestDeepStruct{{Name: "Wood Williams", Phone: "+2332655186949999", Email: "contact@mail.com"}, {Name: "Wood White", Phone: "+233265518694", Email: "contact@example.com"}},
		Allergy:     []int{34359738368, 34359738369},
	}

	myLogger.Println(New().ValidateStruct(request))
}
