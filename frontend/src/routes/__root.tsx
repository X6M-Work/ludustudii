import * as React from 'react'
import { Link, Outlet, createRootRoute } from '@tanstack/react-router'

export const Route = createRootRoute({
  component: RootComponent,
})

// Conditionally load devtools only in development
const TanStackRouterDevtools =
  import.meta.env.PROD
    ? () => null
    : React.lazy(() =>
        import('@tanstack/react-router-devtools').then((res) => ({
          default: res.TanStackRouterDevtools,
        })),
      )

function RootComponent() {
  return (
    <React.Fragment>
      <nav className="flex gap-4 p-6">
        <Link to="/learning-cards">Learning Cards</Link>
        <Link to="/learning-card">Add Card</Link>
      </nav>
      <Outlet />
      <React.Suspense>
        <TanStackRouterDevtools />
      </React.Suspense>      
    </React.Fragment>
  )
}
