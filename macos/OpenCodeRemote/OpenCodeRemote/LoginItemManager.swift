import Foundation
import ServiceManagement

@available(macOS 13.0, *)
enum LoginItemManager {
    static let service = SMAppService.mainApp

    static var isEnabled: Bool {
        service.status == .enabled
    }

    static func register() {
        do {
            try service.register()
        } catch {
            NSLog("LoginItemManager.register failed: %@", error.localizedDescription)
        }
    }

    static func unregister() {
        do {
            try service.unregister()
        } catch {
            NSLog("LoginItemManager.unregister failed: %@", error.localizedDescription)
        }
    }

    static var isInstalledInApplications: Bool {
        let bundlePath = Bundle.main.bundlePath
        return bundlePath.hasPrefix("/Applications/") || bundlePath.hasPrefix("/Users/")
            && bundlePath.contains("/Applications/")
    }
}