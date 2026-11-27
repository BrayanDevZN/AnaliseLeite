import type { LucideIcon } from "lucide-react"

import { TiltCard } from "@/components/tilt-card"
import { Card, CardContent } from "@/components/ui/card"

/** Cada cartão ganha um tom da paleta: o ícone fica num círculo colorido. */
const TONES = {
  primary: "bg-primary/12 text-primary",
  butter: "bg-butter/35 text-accent-foreground",
  sky: "bg-sky/30 text-chart-3",
  coral: "bg-coral/25 text-chart-4",
  lilac: "bg-lilac/30 text-chart-5",
}

export type Tone = keyof typeof TONES

/** Um número do dashboard, com o ícone e o nome dele. */
export function StatCard({ icon: Icon, label, value, hint, tone = "primary" }: { icon: LucideIcon; label: string; value: string; hint?: string; tone?: Tone }) {
  return (
    <TiltCard>
    <Card size="sm" className="depth">
      <CardContent className="flex flex-col gap-1">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <span className={`flex size-7 items-center justify-center rounded-full ${TONES[tone]}`}>
            <Icon className="size-4" aria-hidden="true" />
          </span>
          {label}
        </div>
        <span className="text-2xl font-semibold tracking-tight">{value}</span>
        {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      </CardContent>
    </Card>
    </TiltCard>
  )
}
