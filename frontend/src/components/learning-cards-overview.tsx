import { useCallback, useEffect, useState } from 'react'
import { Link } from '@tanstack/react-router'
import { useVirtualizer, type VirtualizerOptions } from '@tanstack/react-virtual'
import { useLearningCards, type LearningCard } from '@/api/learning-cards'
import { LearningCardView } from '@/components/learning-card-view'
import { Button } from '@/components/ui/button'
import {
  Carousel,
  CarouselContent,
  CarouselItem,
  CarouselNext,
  CarouselPrevious,
  useCarousel,
} from '@/components/ui/carousel'
import { FieldError } from '@/components/ui/field'

export function LearningCardsOverview() {
  const cards = useLearningCards()

  return (
    <main className="mx-auto max-w-3xl p-6">
      <div className="mb-6 flex items-center justify-between gap-4">
        <h1 className="text-xl font-semibold">Learning Cards</h1>
        <Link to="/learning-card" className="text-sm text-primary underline underline-offset-4">
          Add Card
        </Link>
      </div>

      {cards.isPending && <p role="status" className="text-muted-foreground">Loading cards...</p>}
      {cards.isError && (
        <div className="space-y-3" role="alert">
          <FieldError>{cards.error.message}</FieldError>
          <Button variant="outline" onClick={() => cards.refetch()} disabled={cards.isFetching}>
            {cards.isFetching ? 'Retrying...' : 'Try again'}
          </Button>
        </div>
      )}
      {cards.isSuccess && (
        cards.data.length === 0 ? (
          <p className="text-muted-foreground">No learning cards yet. Add a card to get started.</p>
        ) : (
          <Carousel opts={{ align: 'start', loop: false }} aria-label="Learning cards">
            <VirtualCards cards={cards.data} />
          </Carousel>
        )
      )}
    </main>
  )
}

function VirtualCards({ cards }: { cards: LearningCard[] }) {
  const { api } = useCarousel()
  const [width, setWidth] = useState(0)
  const [selectedIndex, setSelectedIndex] = useState(0)

  useEffect(() => {
    if (!api) return

    const update = () => {
      setWidth(api.rootNode().clientWidth)
      setSelectedIndex(api.selectedScrollSnap())
    }
    update()
    api.on('select', update)
    api.on('reInit', update)

    const observer = new ResizeObserver(update)
    observer.observe(api.rootNode())
    return () => {
      observer.disconnect()
      api.off('select', update)
      api.off('reInit', update)
    }
  }, [api])

  // Embla moves the track with transforms, so native scrollLeft stays at zero.
  const observeOffset = useCallback<
    VirtualizerOptions<HTMLElement, HTMLDivElement>['observeElementOffset']
  >((_instance, callback) => {
    if (!api) return

    const update = () => {
      const progress = Math.max(0, Math.min(1, api.scrollProgress()))
      callback(progress * Math.max(0, cards.length - 1) * api.rootNode().clientWidth, true)
    }
    const settle = () => {
      const progress = Math.max(0, Math.min(1, api.scrollProgress()))
      callback(progress * Math.max(0, cards.length - 1) * api.rootNode().clientWidth, false)
    }
    settle()
    api.on('scroll', update)
    api.on('settle', settle)
    api.on('reInit', settle)
    return () => {
      api.off('scroll', update)
      api.off('settle', settle)
      api.off('reInit', settle)
    }
  }, [api, cards.length])

  const virtualizer = useVirtualizer<HTMLElement, HTMLDivElement>({
    count: cards.length,
    getScrollElement: () => api?.rootNode() ?? null,
    horizontal: true,
    estimateSize: () => width,
    getItemKey: (index) => cards[index]!.id,
    observeElementOffset: observeOffset,
    overscan: 1,
    enabled: width > 0,
  })

  useEffect(() => {
    virtualizer.measure()
  }, [virtualizer, width])

  const renderedIndexes = new Set(virtualizer.getVirtualIndexes())
  // Keep the selected card mounted while the viewport is first being measured.
  renderedIndexes.add(selectedIndex)

  return (
    <>
      <CarouselContent className="ml-0 items-start">
        {cards.map((card, index) => (
          // Embla needs every slide's geometry, but only nearby card views are mounted.
          <CarouselItem
            key={card.id}
            className="pl-0"
            aria-label={`${index + 1} of ${cards.length}`}
            aria-hidden={index !== selectedIndex}
            inert={index !== selectedIndex}
          >
            {renderedIndexes.has(index) && <LearningCardView card={card} />}
          </CarouselItem>
        ))}
      </CarouselContent>
      <div className="mt-4 flex items-center justify-center gap-4">
        <CarouselPrevious className="static inset-auto m-0" />
        <p role="status" aria-live="polite" aria-atomic="true" className="text-sm text-muted-foreground">
          {selectedIndex + 1} / {cards.length}
        </p>
        <CarouselNext className="static inset-auto m-0" />
      </div>
    </>
  )
}
