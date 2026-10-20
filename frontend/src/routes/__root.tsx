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
      <nav>
        <Link to="/learning-card">Learning Card</Link>
      </nav>
      <Outlet />
      <React.Suspense>
        <TanStackRouterDevtools />
      </React.Suspense>      
    </React.Fragment>
  )
}
