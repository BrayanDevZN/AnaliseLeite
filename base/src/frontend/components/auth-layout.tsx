import type { ReactNode } from "react"

import mascot from "@/assets/mascote.webp"
import { CowSpots } from "@/components/cow-spots"
import { ThemeToggle } from "@/components/theme-toggle"

/** Moldura das telas de entrar e cadastrar: mascote no topo e manchas de vaca ao fundo. */
export function AuthLayout({ title, description, children }: { title: string; description: string; children: ReactNode }) {
  return (
    <div className="relative flex min-h-dvh flex-col items-center justify-center overflow-hidden px-4 py-8 pt-[max(2rem,env(safe-area-inset-top))] pb-[max(2rem,env(safe-area-inset-bottom))]">
      <CowSpots className="pointer-events-none absolute -top-10 -right-24 size-[28rem] rotate-12 text-primary/[0.07]" />
      <CowSpots className="pointer-events-none absolute -bottom-24 -left-28 size-[26rem] -rotate-12 text-primary/[0.07]" />

      <div className="absolute top-[max(0.75rem,env(safe-area-inset-top))] right-3">
        <ThemeToggle />
      </div>

      <main className="relative flex w-full max-w-sm flex-col items-center gap-6">
        <div className="flex flex-col items-center gap-3 text-center">
          <img src={mascot} alt="Mascote: uma vaca de óculos escuros com um balde de leite" width={640} height={640} className="size-32 rounded-full ring-4 ring-primary/30 sm:size-36" />
          <div className="flex flex-col gap-1">
            <p className="font-heading text-sm font-medium tracking-wide text-primary uppercase">Análise Leite</p>
            <h1 className="font-heading text-2xl font-semibold tracking-tight">{title}</h1>
            <p className="text-sm text-muted-foreground">{description}</p>
          </div>
        </div>
        {children}
      </main>
    </div>
  )
}
