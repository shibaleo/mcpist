import { clerkMiddleware } from "@clerk/nextjs/server"

export const proxy = clerkMiddleware()

export const config = {
  matcher: [
    '/((?!_next/static|_next/image|favicon.ico|.*\\.(?:svg|png|jpg|jpeg|gif|webp)$).*)',
  ],
}
