package app

import (
	"learning-roadmap/internal/modules/courses"
	"learning-roadmap/internal/modules/roadmap"
	"learning-roadmap/internal/modules/settings"
)

type Settings = settings.Settings

type Direction = roadmap.Direction
type RoadmapNode = roadmap.RoadmapNode
type NodeDependency = roadmap.NodeDependency

type Course = courses.Course
type CourseSummary = courses.CourseSummary
type CourseModule = courses.CourseModule
type CourseResource = courses.CourseResource
type CourseModuleRoadmapLink = courses.CourseModuleRoadmapLink
type courseImportPreviewResponse = courses.ImportPreviewResponse

const (
	courseModuleNotStarted = courses.CourseModuleNotStarted
	courseModuleInProgress = courses.CourseModuleInProgress
)
