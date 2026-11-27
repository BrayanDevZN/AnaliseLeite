import type { ReactNode } from "react"
import { ArrowLeftIcon } from "lucide-react"
import { Link } from "react-router"

import { buttonVariants } from "@/components/ui/button"
import { cn } from "@/lib/utils"

interface Props {
  title: string
  description?: string
  /** Mostra uma seta de voltar levando para esta rota. */
  back?: string
  /** Botões de ação, ao lado do título. */
  actions?: ReactNode
}

export function PageHeader({ title, description, back, actions }: Props) {
  return (
    <header className="flex flex-col gap-3">
      {back && (
        <Link to={back} className={cn(buttonVariants({ variant: "ghost", size: "sm" }), "w-fit -ml-2")}>
          <ArrowLeftIcon data-icon="inline-start" />
          Voltar
        </Link>
      )}
      <div className="flex items-start justify-between gap-3">
        <div className="flex min-w-0 flex-col gap-1">
          <h1 className="font-heading text-2xl font-semibold tracking-tight break-words">{title}</h1>
          {description && <p className="text-sm text-muted-foreground">{description}</p>}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
    </header>
  )
}
