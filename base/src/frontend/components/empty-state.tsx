import type { ReactNode } from "react"

import mascot from "@/assets/mascote.webp"
import { Empty, EmptyContent, EmptyDescription, EmptyHeader, EmptyMedia, EmptyTitle } from "@/components/ui/empty"

/** Tela vazia (nada cadastrado ainda), com o mascote e, se houver, o botão para começar. */
export function EmptyState({ title, description, action }: { title: string; description: string; action?: ReactNode }) {
  return (
    <Empty className="border border-dashed">
      <EmptyHeader>
        <EmptyMedia>
          <img src={mascot} alt="" width={96} height={96} className="size-20 rounded-full opacity-90" />
        </EmptyMedia>
        <EmptyTitle>{title}</EmptyTitle>
        <EmptyDescription>{description}</EmptyDescription>
      </EmptyHeader>
      {action && <EmptyContent>{action}</EmptyContent>}
    </Empty>
  )
}
