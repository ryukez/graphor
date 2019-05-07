# graphor
Golang Object-Relational mapper for dgraph (https://dgraph.io/), inspired by Laravel-Eloquent (https://readouble.com/laravel/5.7/ja/eloquent.html).

## Dependency
- dgraph-io/dgo (https://github.com/dgraph-io/dgo)
- google.golang.org/grpc (https://github.com/grpc/grpc-go)

## Install
`go get github.com/nosukeru/graphor`

## Usage
### 1. Define your domain model

```golang
type ImageModel struct {
	Url string `json:"url"`
}

type UserModel struct {
	Uid           string `json:"uid"`
	Id            string `json:"id"`
	Name          string `json:"name"`
	Biography     string `json:"biography"`
	Age           int    `json:"age"`
	Icon          *Image `json:"icon"`
	IsFollowing   bool   `json:"is_following"`
	IsFollowed    bool   `json:"is_followed"`
	FollowCount   int    `json:"follow_count"`
	FollowerCount int    `json:"follower_count"`
	CreatedAt     int    `json:"created_at"`
	UpdatedAt     int    `json:"updated_at"`
}
```

### 2. Define graphor model & schema

By embedding `graphor.ModelProperty`, you can use your model as graphor model.

```golang
import "github.com/nosukeru/graphor"

type Image struct {
	graphor.ModelProperty
	ImageModel
}

type User struct {
	graphor.ModelProperty
	UserModel
}

```

In addition, you should define model schema to tell graphor how to handle your model & relation.

```golang
func ImageSchema() graphor.Schema {
	return graphor.Schema{
		Tag: 2,
		Fields: []string{
			"url",
		},
	}
}

func UserSchema() graphor.Schema {
	return graphor.Schema{
		Tag: 1,
		Fields: []string{
			"id",
			"name",
			"biography",
			"age",
		},
		Booleans: map[string]graphor.Boolean{
			"is_following": graphor.Boolean{
				Edge:   "~follow",
				Filter: "uid(<#{login_uid}>)",
			},
			"is_followed": eloquent.Boolean{
				Edge:   "follow",
				Filter: "uid(<#{login_uid}>)",
			},
		},
		Relations: map[string]graphor.RelationSchema{
			"icon": graphor.RelationSchema{
				Edge:       "has_icon",
				HasMany:    false,
				Include:    true,
				SchemaFunc: ImageSchema,
			},
			"follows": graphor.RelationSchema{
				Edge:       "follow",
				HasMany:    true,
				Include:    false,
				CountField: "follow_count",
				Facets: map[string]eloquent.Facet{
					"followed_at": eloquent.Facet{
						Edge: "at",
					},
				},
				SchemaFunc: UserSchema,
			},
			"followers": graphor.RelationSchema{
				Edge:       "~follow",
				HasMany:    true,
				Include:    false,
				CountField: "follower_count",
				Facets: map[string]eloquent.Facet{
					"followed_at": eloquent.Facet{
						Edge: "at",
					},
				},
				SchemaFunc: UserSchema,
			},
		},
	}
```

#### Tag
A number to identify model type. You can assign arbitary integer, but numbers shouldn't duplicate between distinct models.

#### Fields
Properties which will be saved in dgraph database. Don't include following properties:

- `uid`, `created_at`, `updated_at`, `deleted_at` (these are auto-saved by graphor)
- booleans (e.g. `is_following`, `is_followed`)
- relation models (e.g. `icon`)
- relation counts (e.g. `follow_count`, `follower_count`)

#### Booleans
Boolean flags for relation which indicates whether filter condition holds or not. Filter should be described in GraphQL+-. See details for GraphQL+- in https://docs.dgraph.io/master/query-language/.

For example, boolean in above example

```golang
// UserSchema
  "is_following": graphor.Boolean{
    Edge:   "~follow",
    Filter: "uid(<#{login_uid}>)",
  },
```

checks if that record has reverse-follow edge where `uid` of source record is `#{login_uid}`. Here, you can use uid of login (current, authenticated) user in form `#{login_uid}` by call `graphor.Auth().SetLoginUid(uid string)` beforehand.

#### Relations
Relations which wrap edges in dgraph database. You should relation features in model schema.

- Edge(string): edge name in dgraph
- HasMany(bool): whether allows multiple records for relation or not
- Include(bool): whether includes relation model into current model or not
- IncludeOptions(string): filter for included relation (e.g. `@facets(orderdesc: at) (first: 3)`)
- CountField(string): set field name for relation count if you want include count
- Facets(map[string]Facet): facet list for relation. `Facet` is a struct which has only `Edge` property at this time. Map key is an arbitary name and Facet.Edge is facet name in dgraph database.
- SchemaFunc: relation model schema function (function which returns `Schema` with no arguments) (e.g. `UserSchema`)

### 3. Add some utility methods

```golang
// ----- Image -----

func NewImage(i ...*ImageModel) *Image {
	image := new(Image)
	if len(i) > 0 && i[0] != nil {
		image.Image = *i[0]
	}
	return image
}

// Mutation Utilities
func (image *Image) Save() {
	graphor.Save(image, ImageSchema())
}

func (image *Image) Delete() {
	graphor.Delete(image)
}

// ----- User -----

func NewUser(u ...*UserModel) *User {
	user := new(User)
	if len(u) > 0 && u[0] != nil {
		user.User = *u[0]
		user.SetUid(u[0].Uid)
	}
	return user
}

// Query Utilities
func Users() graphor.Query {
	schema := UserSchema()
	return graphor.BuildQuery(schema)
}

func AsUser(q graphor.Query) (*User, error) {
	user := new(User)

	data, err := q.First()
	if data == nil {
		return nil, err
	}

	graphor.Init(user, data)
	return user, nil
}

func AsUsers(q graphor.Query) ([]*User, error) {
	users := []*User{}

	dataList, err := q.All()
	if dataList == nil {
		return nil, err
	}

	for _, data := range dataList {
		user := new(User)
		graphor.Init(user, data)
		users = append(users, user)
	}

	return users, nil
}

// Mutation Utilities
func (user *User) Save() {
	graphor.Save(user, UserSchema())
}

func (user *User) Delete() {
	graphor.Delete(user)
}

// Relation Utilities
func (user *User) HasIcon() graphor.Relation {
	return graphor.BuildRelation(user, UserSchema().Relations["icon"])
}

func (user *User) HasFollows() graphor.Relation {
	return graphor.BuildRelation(user, UserSchema().Relations["follows"])
}

func (user *User) HasFollowers() graphor.Relation {
	return graphor.BuildRelation(user, UserSchema().Relations["followers"])
}
```

### 4. Alter dgraph schema

```golang
func Alter() error {
	schemaList := []graphor.Schema{
		ImageSchema(),
		UserSchema(),
	}

	migrationBody := graphor.BaseMigrations(schemaList)

	// User
	migrationBody += `
		id: string @index(exact, trigram) .
		name: string @index(trigram) .
		biography: string @index(trigram) .
		age: int @index(int) .
	`

	// Image
	migrationBody += `
		url: string .
	`
	
	return graphor.MigrateDatabase(migrationBody)
}
```

That's all! Now you can use query builder for your custom model.

### Save or Update

```golang
func saveUser(id, name, biography string, iconModel ImageModel) (*UserModel, error) {
	user := NewUser() // For update -> user := NewUser(userModel)
	err := graphor.Mutate(func() error {
		user.Id = id
		user.Name = name
		user.Biography = biography
		user.Save()

		icon := NewImage(iconModel)
		icon.Save()

		user.HasIcon().Set(icon) // For HasOne relation use Relation.Set, and for HasMany relation use Relation.Add instead.

		// Delete old icon
		icon = NewImage(user.Icon) // If user.Icon is null, nothing done
		graphor.HardDelete(icon) // SoftDelete for graphor.Delete, and HardDelete for graphor.HardDelete

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Get updated model
	user, err = AsUser(Users().Identify(user.GetUid())) // Idetify single record by uid
	if err != nil {
		return nil, err
	}

	return &user.User, nil
}
```

### Get Followers

```golang
func GetFollowers(userModel UserModel) ([]*UserModel, error) {
	graphor.Auth().SetLoginUid("0x1") // Set login user of uid "0x1"

	user := NewUser(userModel)

	users, err := AsUsers(user.HasFollowers().SetSortOption("followed_at", "desc").Take(10))
	if err != nil {
		return nil, err
	}

	userModels := []*UserModel{}
	for _, user := range users {
		userModels = append(userModels, &user.User)
	}
	return userModels, nil
}
```

### Follow

```golang
import "github.com/nosukeru/graphor/timestamp"

func Follow(selfModel *UserModel, userModel *UserModel) error {
	self := NewUser(selfModel)
	user := NewUser(userModel)

	return graphor.Mutate(func() error {
		self.HasFollows().Add(user, map[string]interface{}{
			"followed_at": timestamp.Now(),
		})
		return nil
	})
}
```

### Other Queries

```golang
func Examples() {
	// --- Filter ---

	// You can use all indecies available in GraphQL+- (https://docs.dgraph.io/master/query-language/#indexing). 
	// Note that you should designate index in migration.
	users := AsUsers(Users().Where("age", "lt", 20))
	
	// --- Has ---
	
	// HasNot for reverse filter
	users := AsUsers(Users().Has("deleted_at")) // SoftDeleted users
	
	// --- Or, Regex ---
	regex := "/search_word/"

	query := Users().Or(
		func(q graphor.Query) graphor.Query {
			return q.Regex("id", regex)
		},
		func(q graphor.Query) graphor.Query {
			return q.Regex("name", regex)
		},
		func(q graphor.Query) graphor.Query {
			return q.Regex("biography", regex)
		},
	)

	// --- Get raw data ---
	dataList, err := query.All() // Get all records
	data, err := query.First() // Get first record
	
	// --- Scope ---
	onlyAdults := func(q graphor.Query) graphor.Query { return q.Where("age", "ge", 18) }
	users := AsUsers(Users().Scope(onlyAdults))
	
	// --- Paging ---
	users := Users().Where("age", "lt", 20).Paging(sinceTimestamp, untilTimestamp, count)
	
	// --- Exists ---
	exists, err := Users().Where("id", "eq", "user_id").Exists()
	
	// --- Relation.Remove / Relation.Clear ---
	users[0].HasFollowers().Remove(follower) // Remove is only allowed for HasOne relation
	users[1].HasFollows().Clear()
}
```

### Raw Query
You can also write raw query language by using `graphor.BuildRawQuery(q string, schema graphor.Schema, args map[string]interface{})`.

```golang
func GetFollows(self *User) ([]*User, err) {
	q = graphor.BuildRawQuery(`
	{
		var(func: uid(<#{uid}>)) {
			follows as #{follow_edge} #{filter} { uid }
		}

		q(func: uid(follows), #{sorting}#{take}) {
			#{body}
		}
	}`, UserSchema(), map[string]interface{}{
		"uid":         self.GetUid(),
		"follow_edge": "follow",
	})
	
	return AsUsers(q.Where("age", "eq", 20).Take(5))
}
```

You can embed some variables by `#{variable}` notation. There are some pre-defined variables:

- filter: auto-generated filters by `Where()`, `Has()`, ...
- sorting, take: sorting key & order, take count which is set by `SetSortOption()`, `Take()`
- body: auto-generated body according to `schema`

Utilizing these variables, you can combine your own complicated query with builder functions.

## Help
If you have problems, please feel free to contact.

- Github: https://github.com/nosukeru
- Twitter: @ey_nosukeru
- Gmail: ey.nosukeru[at]gmail.com
