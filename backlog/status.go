package backlog

func GetStatusByCode(statusCode string) string {
	switch statusCode {
	case "l":
		return "landed"
	case "f":
		return "flying"
	case "g":
		return "gate"
	case "h":
		return "hangar"
	}
	return "unknown"
}

func GetStatusDescriptionByCode(statusCode string) string {
	switch statusCode {
	case "l":
		return "landed"
	case "f":
		return "in flight"
	case "g":
		return "at the gate"
	case "h":
		return "in hangar"
	}
	return "unknown"
}
