package spine

type Skeleton struct {
	Hash       string
	Version    string
	X          float32
	Y          float32
	Width      float32
	Height     float32
	FPS        float32
	ImagesPath string
	AudioPath  string
	Bones      []Bone
	Slots      []Slot
	IK         []IKConstraint
	Transform  []TransformConstraint
	Path       []PathConstraint
	Skins      []Skin
	Events     []Event
	Animations []Animation
}

type Bone struct {
	Name          string
	Parent        int
	Rotation      float32
	X             float32
	Y             float32
	ScaleX        float32
	ScaleY        float32
	ShearX        float32
	ShearY        float32
	Length        float32
	TransformMode int
	SkinRequired  bool
	Color         uint32
}

type Slot struct {
	Name       string
	Bone       int
	Color      uint32
	DarkColor  uint32
	HasDark    bool
	Attachment string
	BlendMode  int
}

type IKConstraint struct {
	Name          string
	Order         int
	SkinRequired  bool
	Bones         []int
	Target        int
	Mix           float32
	Softness      float32
	BendDirection int
	Compress      bool
	Stretch       bool
	Uniform       bool
}

type TransformConstraint struct {
	Name           string
	Order          int
	SkinRequired   bool
	Bones          []int
	Target         int
	Local          bool
	Relative       bool
	OffsetRotation float32
	OffsetX        float32
	OffsetY        float32
	OffsetScaleX   float32
	OffsetScaleY   float32
	OffsetShearY   float32
	MixRotate      float32
	MixX           float32
	MixY           float32
	MixScaleX      float32
	MixScaleY      float32
	MixShearY      float32
}

type PathConstraint struct {
	Name           string
	Order          int
	SkinRequired   bool
	Bones          []int
	Target         int
	PositionMode   int
	SpacingMode    int
	RotateMode     int
	OffsetRotation float32
	Position       float32
	Spacing        float32
	MixRotate      float32
	MixX           float32
	MixY           float32
}

type Skin struct {
	Name        string
	Bones       []int
	IK          []int
	Transform   []int
	Path        []int
	Attachments []Attachment
}

type AttachmentType int

const (
	AttRegion AttachmentType = iota
	AttBoundingBox
	AttMesh
	AttLinkedMesh
	AttPath
	AttPoint
	AttClipping
)

func (t AttachmentType) String() string {
	names := [...]string{"region", "boundingbox", "mesh", "linkedmesh", "path", "point", "clipping"}
	if int(t) >= 0 && int(t) < len(names) {
		return names[t]
	}
	return "unknown"
}

type Attachment struct {
	Slot          int
	Name          string
	Type          AttachmentType
	Path          string
	X             float32
	Y             float32
	Rotation      float32
	ScaleX        float32
	ScaleY        float32
	Width         float32
	Height        float32
	Color         uint32
	Vertices      Vertices
	UVs           []float32
	Triangles     []uint16
	HullLength    int
	Edges         []uint16
	SkinName      string
	Parent        string
	InheritDeform bool
	Closed        bool
	ConstantSpeed bool
	Lengths       []float32
	EndSlot       int
	Sequence      *Sequence
}

type Sequence struct {
	Count      int
	Start      int
	Digits     int
	SetupIndex int
}

type Vertices struct {
	Count    int
	Weighted bool
	Bones    []int
	Values   []float32
}

type Event struct {
	Name      string
	Int       int
	Float     float32
	String    string
	HasAudio  bool
	AudioPath string
	Volume    float32
	Balance   float32
}

type Animation struct {
	Name      string
	Duration  float32
	Timelines []Timeline
}

type Timeline struct {
	Kind       string
	Target     int
	Attachment string
	Frames     int
	Times      []float32
}

type Atlas struct {
	Pages []Page
}

type Page struct {
	Name      string
	Width     int
	Height    int
	Format    string
	MinFilter string
	MagFilter string
	Repeat    string
	PMA       bool
	Scale     float32
	Regions   []Region
}

type Region struct {
	Page       int
	Name       string
	X          int
	Y          int
	Width      int
	Height     int
	OffsetX    int
	OffsetY    int
	OrigWidth  int
	OrigHeight int
	Rotate     bool
	Degrees    int
	Index      int
	Splits     []int
	Pads       []int
}
