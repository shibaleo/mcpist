"use client"

import { Sheet, SheetContent, SheetTrigger, SheetTitle } from "@/components/ui/sheet"
import * as VisuallyHidden from "@radix-ui/react-visually-hidden"
import { PanelLeft } from "lucide-react"
import { Sidebar } from "./sidebar"
import { useState } from "react"

export function MobileHeader() {
  const [open, setOpen] = useState(false)

  return (
    <div className="md:hidden fixed z-50">
      <Sheet open={open} onOpenChange={setOpen}>
        <SheetTrigger asChild>
          <button className="fixed top-4 left-3 z-40 h-8 w-8 flex items-center justify-center rounded-md text-muted-foreground hover:text-foreground hover:bg-sidebar-accent/50 transition-colors bg-transparent">
            <PanelLeft className="h-4 w-4" />
            <span className="sr-only">メニューを開く</span>
          </button>
        </SheetTrigger>
        <SheetContent side="left" className="p-0 w-64" hideCloseButton>
          <VisuallyHidden.Root>
            <SheetTitle>ナビゲーションメニュー</SheetTitle>
          </VisuallyHidden.Root>
          <Sidebar onClose={() => setOpen(false)} />
        </SheetContent>
      </Sheet>
    </div>
  )
}
