import type { LucideIcon } from "lucide-react"

import { Card, CardContent } from "@/components/ui/card"

/** Um número do dashboard, com o ícone e o nome dele. */
export function StatCard({ icon: Icon, label, value, hint }: { icon: LucideIcon; label: string; value: string; hint?: string }) {
  return (
    <Card size="sm">
      <CardContent className="flex flex-col gap-1">
        <div className="flex items-center gap-2 text-xs text-muted-foreground">
          <Icon className="size-4 text-primary" aria-hidden="true" />
          {label}
        </div>
        <span className="text-2xl font-semibold tracking-tight">{value}</span>
        {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
      </CardContent>
    </Card>
  )
}
