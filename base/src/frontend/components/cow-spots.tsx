/** Manchas de vaca como detalhe de fundo. Decorativo: não captura toque nem aparece para leitores de tela. */
export function CowSpots({ className }: { className?: string }) {
  return (
    <svg
      aria-hidden="true"
      viewBox="0 0 400 400"
      preserveAspectRatio="xMidYMid slice"
      className={className}
      fill="currentColor"
    >
      <path d="M52 40c40-26 96-18 118 20 20 34-6 62-34 84-30 24-64 44-92 20-32-28-28-96 8-124Z" />
      <path d="M268 18c36-14 92 8 104 46 10 34-18 52-44 64-30 14-70 28-92 2-24-28-8-96 32-112Z" />
      <path d="M300 210c34-18 80 4 88 44 8 38-26 66-58 78-34 12-72 4-82-30-10-34 14-76 52-92Z" />
      <path d="M26 240c30-20 76-8 90 26 12 30-8 56-34 70-32 18-72 6-82-30-8-30 4-48 26-66Z" />
      <path d="M170 318c26-16 66-4 74 26 8 28-14 48-40 54-30 8-62-8-66-36-2-22 10-34 32-44Z" />
    </svg>
  )
}
