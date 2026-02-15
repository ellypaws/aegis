import { Link } from "react-router-dom";
import { UI } from "../constants";
import { cn } from "../lib/utils";

export function NotFound() {
  return (
    <div className={UI.page}>
      <div className={UI.max}>
        <div className="flex flex-col items-center justify-center min-h-[60vh]">
          <div className={cn("text-center p-8 max-w-lg", UI.cardRed)}>
            <div className="mb-6">
              <span className="text-8xl font-black text-red-400">404</span>
            </div>
            
            <h1 className="text-2xl font-black text-zinc-900 mb-4">
              Page Not Found
            </h1>
            
            <p className="text-zinc-600 mb-8 font-medium">
              Oops! The page you're looking for seems to have wandered off into the void. 
              Maybe it went to grab a snack?
            </p>

            <div className="flex flex-col sm:flex-row gap-3 justify-center">
              <Link
                to="/"
                className={cn(UI.button, UI.btnGreen)}
              >
                Back to Gallery
              </Link>
              
              <button
                type="button"
                onClick={() => window.history.back()}
                className={cn(UI.button, UI.btnYellow)}
              >
                Go Back
              </button>
            </div>
          </div>

          <div className="mt-8 text-center">
            <div className={cn("inline-block px-4 py-2", UI.soft)}>
              <span className={UI.label}>Lost?</span>
              <p className="text-sm text-zinc-600 mt-1">
                Try checking the URL or navigate back to the gallery
              </p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
