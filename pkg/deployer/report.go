package deployer

import (
	"sort"
	"strconv"
	"strings"
	"time"
)


const ReportColumnsCount = 7

type SingleReport struct {
	Name                  string

	InstancesTotal        int
	InstancesDeployed     int
	InstancesFailed       int
	InstancesNotAttempted int

	TimeSpentOnPacking    time.Duration
	TimeSpentOnDeploy     time.Duration

	packingStarted time.Time
	deployStarted time.Time
}

func (r *SingleReport) InstancePackingStarted() {
	r.packingStarted = time.Now()
	r.InstancesTotal++
	r.InstancesNotAttempted++
}

func (r *SingleReport) InstancePackingDone() {
	r.TimeSpentOnPacking += time.Since(r.packingStarted)
}

func (r *SingleReport) HostPackingStarted() {
	r.packingStarted = time.Now()
}

func (r *SingleReport) HostPackingDone() {
	r.TimeSpentOnPacking += time.Since(r.packingStarted)
}

func (r *SingleReport) DeployStarted() {
	r.deployStarted = time.Now()
}

func (r *SingleReport) DeployDone() {
	r.TimeSpentOnDeploy += time.Since(r.deployStarted)
}

func (r *SingleReport) InstanceDeployStarted() {
	r.InstancesNotAttempted--
}

func (r *SingleReport) InstanceDeployDone(ok bool) {
	if ok {
		r.InstancesDeployed++
	} else {
		r.InstancesFailed++
	}
}

func (r *SingleReport) ToColumns() []string {
	cols := make([]string, 0, ReportColumnsCount)
	cols = append(cols, r.Name)
	cols = append(cols, strconv.Itoa(r.InstancesTotal))
	cols = append(cols, strconv.Itoa(r.InstancesDeployed))
	cols = append(cols, strconv.Itoa(r.InstancesFailed))
	cols = append(cols, strconv.Itoa(r.InstancesNotAttempted))
	cols = append(cols, r.TimeSpentOnPacking.String())
	cols = append(cols, r.TimeSpentOnDeploy.String())
	return cols
}

type Report struct {
	SummaryAndHostReps []*SingleReport
}

func NewReport() *Report {
	r := &Report{}
	r.CreateHostReport("Summary")
	return r
}

func (r *Report) CreateHostReport(name string) *SingleReport {
	sr := &SingleReport{Name: name}
	r.SummaryAndHostReps = append(r.SummaryAndHostReps, sr)
	return sr
}


func (r *Report) String() string {
	summary := r.SummaryAndHostReps[0]
	for i := 1; i < len(r.SummaryAndHostReps); i++ {
		summary.InstancesTotal += r.SummaryAndHostReps[i].InstancesTotal
		summary.InstancesDeployed += r.SummaryAndHostReps[i].InstancesDeployed
		summary.InstancesFailed += r.SummaryAndHostReps[i].InstancesFailed
		summary.InstancesNotAttempted += r.SummaryAndHostReps[i].InstancesNotAttempted
		summary.TimeSpentOnDeploy += r.SummaryAndHostReps[i].TimeSpentOnDeploy
		summary.TimeSpentOnPacking += r.SummaryAndHostReps[i].TimeSpentOnPacking
	}

	b := strings.Builder{}

	data := make([][]string, len(r.SummaryAndHostReps))
	for i := range data {
		data[i] = r.SummaryAndHostReps[i].ToColumns()
	}
	colLens := make([]int, ReportColumnsCount)

	for j:=0; j<ReportColumnsCount; j++ {
		for i := range data {
			if colLens[j] < len(data[i][j]) {
				colLens[j] = len(data[i][j])
			}
		}
	}

	separators := []string{
		": instances: ",
		" deployed: ",
		" failed: ",
		" not attempted: ",
		" spent on packing: ",
		" spent on deploy: ",
	}

	sort.Slice(data[1:], func (i, j int) bool {
		return data[i+1][0] < data[j+1][0]
	})

	lineLen := 0
	for _, maxLen := range colLens {
		lineLen += maxLen
	}
	for _, sep := range separators {
		lineLen += len(sep)
	}

	for i:=0; i<lineLen; i++ {
		b.WriteByte('-')
	}
	b.WriteByte('\n')

	for i := range data {
		for j:=0; j < ReportColumnsCount; j++ {
			chars, _ := b.WriteString(data[i][j])
			for k := 0; k < colLens[j] - chars; k++ {
				b.WriteByte(' ')
			}
			if j != ReportColumnsCount - 1 {
				b.WriteString(separators[j])
			}
		}
		b.WriteByte('\n')
	}

	return b.String()
}