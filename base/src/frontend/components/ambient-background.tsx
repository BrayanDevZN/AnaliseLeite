import type { CSSProperties } from "react"

/** As manchas de cor: uma de cada tom da paleta (verde, manteiga, céu, lilás, coral), cada uma com seu ritmo. */
const BLOBS = [
  { color: "var(--primary)", top: "-12%", left: "-10%", size: "34rem", opacity: 0.3, anim: "blob-drift-a 26s ease-in-out infinite" },
  { color: "var(--butter)", top: "-14%", left: "62%", size: "32rem", opacity: 0.42, anim: "blob-drift-b 31s ease-in-out infinite" },
  { color: "var(--sky)", top: "55%", left: "70%", size: "30rem", opacity: 0.36, anim: "blob-drift-a 35s ease-in-out -8s infinite" },
  { color: "var(--lilac)", top: "58%", left: "-12%", size: "28rem", opacity: 0.32, anim: "blob-drift-b 29s ease-in-out -4s infinite" },
  { color: "var(--coral)", top: "28%", left: "36%", size: "22rem", opacity: 0.2, anim: "blob-drift-a 38s ease-in-out -14s infinite" },
]

/** Bolhas de leite que sobem devagar: posição, tamanho, duração e atraso variados. */
const BUBBLES = [
  { left: "6%", size: 10, duration: 24, delay: -2, dx: 30, o: 0.5 },
  { left: "17%", size: 18, duration: 32, delay: -14, dx: -24, o: 0.38 },
  { left: "29%", size: 8, duration: 21, delay: -7, dx: 18, o: 0.55 },
  { left: "41%", size: 14, duration: 29, delay: -20, dx: -30, o: 0.4 },
  { left: "53%", size: 22, duration: 38, delay: -11, dx: 26, o: 0.3 },
  { left: "64%", size: 9, duration: 23, delay: -4, dx: -16, o: 0.5 },
  { left: "76%", size: 16, duration: 30, delay: -17, dx: 22, o: 0.4 },
  { left: "88%", size: 11, duration: 26, delay: -9, dx: -20, o: 0.5 },
  { left: "95%", size: 20, duration: 35, delay: -25, dx: -28, o: 0.3 },
]

/**
 * Fundo vivo, fixo atrás de tudo: manchas de cor desfocadas que derivam bem devagar e bolhas de leite subindo.
 * Só `transform` e `opacity` animam (barato para o celular). Com "reduzir movimento" ligado, fica parado.
 */
export function AmbientBackground() {
  return (
    <div aria-hidden="true" className="pointer-events-none fixed inset-0 -z-10 overflow-hidden">
      {BLOBS.map((blob, index) => (
        <span
          key={index}
          className="ambient-blob absolute rounded-full blur-3xl will-change-transform dark:opacity-60"
          style={
            {
              top: blob.top,
              left: blob.left,
              width: blob.size,
              height: blob.size,
              background: `radial-gradient(circle at 35% 35%, ${blob.color}, transparent 70%)`,
              opacity: blob.opacity,
              animation: blob.anim,
            } as CSSProperties
          }
        />
      ))}

      {BUBBLES.map((bubble, index) => (
        <span
          key={index}
          className="ambient-bubble absolute bottom-[-30px] rounded-full will-change-transform"
          style={
            {
              left: bubble.left,
              width: bubble.size,
              height: bubble.size,
              "--dx": `${bubble.dx}px`,
              "--o": bubble.o,
              background: "radial-gradient(circle at 30% 28%, white, color-mix(in oklch, var(--sky) 50%, transparent) 70%)",
              boxShadow: "inset 0 0 6px color-mix(in oklch, white 70%, transparent)",
              animation: `bubble-rise ${bubble.duration}s linear ${bubble.delay}s infinite`,
            } as CSSProperties
          }
        />
      ))}
    </div>
  )
}
