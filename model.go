package graphor

type Model interface {
	GetUid() string
	SetUid(uid string)
	GetCreatedAt() int
	setCreatedAt(timestamp int)
	GetUpdatedAt() int
	setUpdatedAt(timestamp int)
	GetDeletedAt() int
	setDeletedAt(timestamp int)
	GetData() QueryData
	setData(data QueryData)
	isEmpty() bool
	isNew() bool
	isSaved() bool
}

type ModelProperty struct {
	__uid       string
	__createdAt int
	__updatedAt int
	__deletedAt int
	__data      map[string]interface{}
}

func (model *ModelProperty) GetUid() string {
	return model.__uid
}

func (model *ModelProperty) SetUid(uid string) {
	model.__uid = uid
}

func (model *ModelProperty) GetCreatedAt() int {
	return model.__createdAt
}

func (model *ModelProperty) setCreatedAt(timestamp int) {
	model.__createdAt = timestamp
}

func (model *ModelProperty) GetUpdatedAt() int {
	return model.__updatedAt
}

func (model *ModelProperty) setUpdatedAt(timestamp int) {
	model.__updatedAt = timestamp
}

func (model *ModelProperty) GetDeletedAt() int {
	return model.__deletedAt
}

func (model *ModelProperty) setDeletedAt(timestamp int) {
	model.__deletedAt = timestamp
}

func (model *ModelProperty) GetData() QueryData {
	return model.__data
}

func (model *ModelProperty) setData(data QueryData) {
	model.__data = data
}

func (model *ModelProperty) isEmpty() bool {
	return model.__uid == ""
}

func (model *ModelProperty) isNew() bool {
	return model.__uid[:2] == "_:"
}

func (model *ModelProperty) isSaved() bool {
	return !model.isEmpty() && !model.isNew()
}

func Init(model Model, data QueryData) {
	model.SetUid(decodeString(data["uid"]))
	model.setCreatedAt(decodeInt(data["created_at"]))
	model.setUpdatedAt(decodeInt(data["updated_at"]))
	model.setData(data)
	cast(data, model)
}
