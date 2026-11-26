import { CircleCheckIcon, CircleIcon } from "lucide-react"

import { PASSWORD_RULES } from "@/lib/validation"
import { cn } from "@/lib/utils"

/** As regras da senha, marcadas conforme a pessoa digita. */
export function PasswordChecklist({ value }: { value: string }) {
  return (
    <ul className="flex flex-col gap-1 text-xs" aria-label="Regras da senha">
      {PASSWORD_RULES.map((rule) => {
        const ok = rule.ok(value)

        return (
          <li key={rule.id} className={cn("flex items-center gap-1.5", ok ? "text-success" : "text-muted-foreground")}>
            {ok ? <CircleCheckIcon className="size-3.5" aria-hidden="true" /> : <CircleIcon className="size-3.5" aria-hidden="true" />}
            {rule.label}
            <span className="sr-only">{ok ? " (cumprida)" : " (falta)"}</span>
          </li>
        )
      })}
    </ul>
  )
}
