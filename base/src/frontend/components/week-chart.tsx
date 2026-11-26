import { cn } from "@/lib/utils"
import type { DayStats } from "@/lib/types"

const weekday = new Intl.DateTimeFormat("pt-BR", { weekday: "short" })

/** "2025-09-01" como data LOCAL (new Date("2025-09-01") seria meia-noite UTC e podia mostrar o dia anterior). */
function localDate(date: string): Date {
  const [year, month, day] = date.split("-").map(Number)

  return new Date(year, month - 1, day)
}

/** As análises dos últimos 7 dias, uma barra por dia. A última é hoje. */
export function WeekChart({ days }: { days: DayStats[] }) {
  const max = Math.max(1, ...days.map((day) => day.analyses))
  const label = days.map((day) => `${weekday.format(localDate(day.date))}: ${day.analyses}`).join(", ")

  return (
    <div className="flex h-32 items-stretch gap-2" role="img" aria-label={`Análises nos últimos 7 dias. ${label}`}>
      {days.map((day, index) => {
        const today = index === days.length - 1

        return (
          <div key={day.date} className="flex min-w-0 flex-1 flex-col items-center gap-1">
            <span className="h-4 text-xs font-medium text-muted-foreground">{day.analyses || ""}</span>
            <div className="flex w-full flex-1 items-end">
              <div
                className={cn("w-full rounded-t-md", day.analyses > 0 ? (today ? "bg-primary" : "bg-primary/60") : "bg-muted")}
                style={{ height: `${day.analyses > 0 ? Math.max((day.analyses / max) * 100, 10) : 4}%` }}
              />
            </div>
            <span className={cn("text-[0.7rem] capitalize", today ? "font-semibold text-foreground" : "text-muted-foreground")}>
              {weekday.format(localDate(day.date)).replace(".", "")}
            </span>
          </div>
        )
      })}
    </div>
  )
}
