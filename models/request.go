package models

type RequestStatus string

const (
	Pending  RequestStatus = "pending"
	Approved RequestStatus = "approved"
	Rejected RequestStatus = "rejected"
)

var RequestStatuses = []RequestStatus{Pending, Approved, Rejected}

type TripType string

const (
	OtherTrip  TripType = "other"
	Meeting    TripType = "meeting"
	Travel     TripType = "travel"
	Conference TripType = "conference"
	Seminar    TripType = "seminar"
)

var TripTypes = []TripType{OtherTrip, Meeting, Travel, Conference, Seminar}

type LeaveType string

const (
	OtherLeave LeaveType = "other"
	Sick       LeaveType = "sick"
	Casual     LeaveType = "casual"
	Maternity  LeaveType = "maternity"
	Paternity  LeaveType = "paternity"
)

var LeaveTypes = []LeaveType{OtherLeave, Sick, Casual, Maternity, Paternity}

type Request struct {
	Base
	Description string        `gorm:"not null;size:512"`
	Status      RequestStatus `gorm:"not null;type:hr_request_status;default:pending"`
	ApproverID  string        `gorm:"type:uuid;default:null"`
	Approver    User          `gorm:"foreignKey:ApproverID;constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	RequesterID string        `gorm:"not null;type:uuid"`
	Requester   User          `gorm:"foreignKey:RequesterID;constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	StartDate   int64         `gorm:"not null"`
	EndDate     int64         `gorm:"not null"`
}

type Leave struct {
	Request
	Type LeaveType `gorm:"not null;type:hr_leave_type"`
}
type Trip struct {
	Request
	Type TripType `gorm:"not null;type:hr_trip_type"`
}
