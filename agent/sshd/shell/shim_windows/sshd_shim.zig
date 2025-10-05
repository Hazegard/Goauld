const std = @import("std");
const windows = std.os.windows;

pub fn main() !void {
    const allocator = std.heap.page_allocator;

    var args_it = try std.process.argsWithAllocator(allocator);
    defer args_it.deinit();

    // Skip shim name
    _ = args_it.next();

    const target_prog = args_it.next() orelse {
        std.debug.print("Usage: sshd_shim <program> [args...]\n", .{});
        return error.InvalidArgs;
    };

    // ✅ Zig 0.15: ArrayList must be initialized with struct literal
    var child_argv = std.array_list.Managed([:0]const u8).init(allocator);
    defer child_argv.deinit();


    try child_argv.append(target_prog);
    while (args_it.next()) |arg| {
        try child_argv.append(arg);
    }

    // ✅ Child API also changed in Zig 0.15
    var child = std.process.Child.init(child_argv.items,allocator);
    //defer child.deinit();

    child.argv = child_argv.items;
    child.stdin_behavior = .Inherit;
    child.stdout_behavior = .Inherit;
    child.stderr_behavior = .Inherit;

    // Windows-specific: create new process group
    // child.options.windows_creation_flags = windows.CREATE_NEW_PROCESS_GROUP;

    const term = try child.spawnAndWait();

    switch (term) {
        .Exited => |code| std.process.exit(code),
        .Signal => |sig| {
            const exit_code: u8 = @intCast(128 + sig);
            std.debug.print("Process terminated by signal {}\n", .{sig});
            std.process.exit(exit_code);
        },
        else => {
            std.debug.print("Process terminated unexpectedly\n", .{});
            std.process.exit(1);
        },
    }
}
