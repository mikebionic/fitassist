package ai

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/mike/fitassist/internal/repository"
)

// BuildHealthContext creates a system prompt section with the user's recent health data.
func BuildHealthContext(ctx context.Context, healthRepo *repository.HealthRepository, userID string) string {
	now := time.Now()
	today := now.Truncate(24 * time.Hour)
	weekAgo := today.AddDate(0, 0, -7)
	monthAgo := today.AddDate(0, 0, -30)
	endOfDay := today.Add(24*time.Hour - time.Second)

	var sections []string

	// Today's summary
	summary, err := healthRepo.GetDashboardSummary(ctx, userID, today)
	if err == nil {
		s := "TODAY'S DATA:\n"
		if summary.StepsToday != nil {
			s += fmt.Sprintf("- Steps: %d", *summary.StepsToday)
			if summary.DistanceToday != nil {
				s += fmt.Sprintf(" (%.1f km)", float64(*summary.DistanceToday)/1000)
			}
			s += "\n"
		}
		if summary.CaloriesToday != nil {
			s += fmt.Sprintf("- Calories: %d kcal\n", *summary.CaloriesToday)
		}
		if summary.SleepLastNight != nil {
			h := *summary.SleepLastNight / 60
			m := *summary.SleepLastNight % 60
			s += fmt.Sprintf("- Sleep last night: %dh %dm\n", h, m)
		}
		if summary.AvgHRToday != nil {
			s += fmt.Sprintf("- Avg heart rate today: %.0f bpm\n", *summary.AvgHRToday)
		}
		if summary.LastHR != nil {
			s += fmt.Sprintf("- Last heart rate: %d bpm\n", *summary.LastHR)
		}
		sections = append(sections, s)
	}

	// Weekly steps
	steps, _ := healthRepo.GetSteps(ctx, userID, weekAgo, endOfDay)
	if len(steps) > 0 {
		s := "STEPS (last 7 days):\n"
		totalSteps := 0
		for _, st := range steps {
			v := 0
			if st.TotalSteps != nil {
				v = *st.TotalSteps
			}
			totalSteps += v
			s += fmt.Sprintf("  %s: %d steps\n", st.Date.Format("Jan 2"), v)
		}
		avg := totalSteps / len(steps)
		s += fmt.Sprintf("  Average: %d steps/day, Total: %d\n", avg, totalSteps)
		sections = append(sections, s)
	}

	// Weekly sleep
	sleeps, _ := healthRepo.GetSleep(ctx, userID, weekAgo, endOfDay)
	if len(sleeps) > 0 {
		s := "SLEEP (last 7 days):\n"
		totalMin := 0
		totalDeep := 0
		for _, sl := range sleeps {
			dur := 0
			if sl.DurationMin != nil {
				dur = *sl.DurationMin
			}
			deep := 0
			if sl.DeepMin != nil {
				deep = *sl.DeepMin
			}
			totalMin += dur
			totalDeep += deep
			bedtime := "—"
			if sl.SleepStart != nil {
				bedtime = sl.SleepStart.Format("15:04")
			}
			s += fmt.Sprintf("  %s: %dh%dm (deep: %dm, bedtime: %s)\n",
				sl.Date.Format("Jan 2"), dur/60, dur%60, deep, bedtime)
		}
		avgMin := totalMin / len(sleeps)
		s += fmt.Sprintf("  Average: %dh%dm/night, Avg deep: %dm\n", avgMin/60, avgMin%60, totalDeep/len(sleeps))
		sections = append(sections, s)
	}

	// Recent workouts (last 30 days)
	workouts, _ := healthRepo.GetWorkouts(ctx, userID, monthAgo, endOfDay)
	if len(workouts) > 0 {
		s := fmt.Sprintf("WORKOUTS (last 30 days): %d total\n", len(workouts))
		limit := len(workouts)
		if limit > 5 {
			limit = 5
		}
		for _, w := range workouts[:limit] {
			dur := 0
			if w.DurationSec != nil {
				dur = *w.DurationSec / 60
			}
			dist := ""
			if w.DistanceM != nil && *w.DistanceM > 0 {
				dist = fmt.Sprintf(", %.1fkm", float64(*w.DistanceM)/1000)
			}
			s += fmt.Sprintf("  %s: %s (%dm%s)\n",
				w.StartedAt.Format("Jan 2"), w.WorkoutType, dur, dist)
		}
		sections = append(sections, s)
	}

	if len(sections) == 0 {
		return "No health data available yet. The user hasn't synced their Mi Fitness data."
	}

	return strings.Join(sections, "\n")
}

const SystemPromptTemplate = `You are a personal AI health and fitness assistant called FitAssist.
You analyze health data from the user's wearable device (Mi Band/Amazfit via Mi Fitness).

Your responsibilities:
1. Provide personalized, actionable health recommendations
2. Identify patterns (e.g., poor sleep → low activity next day)
3. Celebrate progress and achievements
4. Warn about concerning trends (high resting HR, chronic sleep deficit)
5. Suggest specific actions for the next day/week
6. Be encouraging but honest

Communication style:
- Be concise and specific, not generic
- Use data to back up your suggestions
- Format responses with clear structure
- Use simple language, avoid medical jargon

USER'S HEALTH DATA:
%s

Respond in the same language the user writes in.`
