import AppKit
import SwiftUI
import Combine

@main
struct OpenCodeRemoteApp {
    static func main() {
        let delegate = AppDelegate()
        NSApplication.shared.delegate = delegate
        NSApp.setActivationPolicy(.accessory)
        NSApplication.shared.run()
    }
}

final class AppDelegate: NSObject, NSApplicationDelegate {
    private var appState: AppState!
    private var botController: BotController!
    private var statusBarController: StatusBarController!
    private var cancellables: Set<AnyCancellable> = []

    func applicationDidFinishLaunching(_ notification: Notification) {
        appState = AppState()
        botController = BotController { [weak self] status in
            guard let self = self else { return }
            DispatchQueue.main.async {
                self.handleStatusUpdate(status)
            }
        }
        statusBarController = StatusBarController(appState: appState, botController: botController)

        botController.stateSubject
            .receive(on: DispatchQueue.main)
            .sink { [weak self] status in
                self?.appState.status = status
                self?.appState.lastError = self?.botController.lastError
                NotificationCenter.default.post(name: .botStateChanged, object: status)
            }
            .store(in: &cancellables)
    }

    func applicationWillTerminate(_ notification: Notification) {
        botController?.shutdown()
    }

    private func handleStatusUpdate(_ status: BotStatus) {
        switch status {
        case .running:
            appState.startedAt = Date()
            appState.lastError = nil
        case .stopped, .crashed:
            appState.startedAt = nil
            appState.lastError = botController.lastError
        case .stopping:
            break
        case .starting:
            break
        }
    }
}