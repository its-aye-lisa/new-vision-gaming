/*
 * Independent Clean-Room C Reference Kernel for spec/app.spec.json (Tic-Tac-Toe).
 *
 * Authored strictly from spec/app.spec.json invariants without sharing Go code.
 * Compiled independently with BOTH gcc and clang to verify 3-way equivalence
 * against the compiled Go release binary (aegis-app --eval-ttt).
 *
 * CLI interface:
 *   tictactoe_kernel <board> <player> <position> <actor>
 * Output:
 *   <VALID|INVALID> <STATUS> <WINNER|-> <NEXT_TURN|-> <REASON> <BOARD>
 */
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

static char detect_winner(const char *b) {
    static const int lines[8][3] = {
        {0, 1, 2}, {3, 4, 5}, {6, 7, 8},
        {0, 3, 6}, {1, 4, 7}, {2, 5, 8},
        {0, 4, 8}, {2, 4, 6}
    };
    for (int i = 0; i < 8; i++) {
        char c = b[lines[i][0]];
        if (c != '.' && c == b[lines[i][1]] && c == b[lines[i][2]]) {
            return c;
        }
    }
    return 0;
}

static void print_out(int valid, const char *status, const char *winner, const char *next_turn, const char *reason, const char *board) {
    printf("%s %s %s %s %s %s\n",
        valid ? "VALID" : "INVALID",
        status,
        (winner && winner[0]) ? winner : "-",
        (next_turn && next_turn[0]) ? next_turn : "-",
        reason,
        board);
}

int main(int argc, char **argv) {
    if (argc != 5) {
        fprintf(stderr, "usage: tictactoe_kernel <board> <player> <position> <actor>\n");
        return 2;
    }
    const char *board = argv[1];
    const char *player = argv[2];
    int pos = atoi(argv[3]);
    (void)argv[4]; /* TTT-INV-5: Actor MUST NEVER influence game outcome */

    if (strlen(board) != 9) {
        print_out(0, "INVALID", "", "", "ERR_INVALID_BOARD", board);
        return 0;
    }
    int x_count = 0, o_count = 0;
    for (int i = 0; i < 9; i++) {
        if (board[i] == 'X') x_count++;
        else if (board[i] == 'O') o_count++;
        else if (board[i] != '.') {
            print_out(0, "INVALID", "", "", "ERR_INVALID_BOARD", board);
            return 0;
        }
    }

    char expected_turn[2] = {0, 0};
    if (x_count == o_count) expected_turn[0] = 'X';
    else if (x_count == o_count + 1) expected_turn[0] = 'O';
    else {
        print_out(0, "INVALID", "", "", "ERR_INVALID_BOARD_BALANCE", board);
        return 0;
    }

    char existing_winner = detect_winner(board);
    if (existing_winner != 0) {
        char w[2] = {existing_winner, 0};
        print_out(0, "INVALID", w, "", "ERR_GAME_ALREADY_OVER", board);
        return 0;
    }
    if (x_count + o_count == 9) {
        print_out(0, "INVALID", "", "", "ERR_GAME_ALREADY_OVER", board);
        return 0;
    }

    if (strcmp(player, "X") != 0 && strcmp(player, "O") != 0) {
        print_out(0, "INVALID", "", expected_turn, "ERR_INVALID_PLAYER", board);
        return 0;
    }
    if (player[0] != expected_turn[0]) {
        print_out(0, "INVALID", "", expected_turn, "ERR_OUT_OF_TURN", board);
        return 0;
    }
    if (pos < 0 || pos > 8) {
        print_out(0, "INVALID", "", expected_turn, "ERR_OUT_OF_BOUNDS", board);
        return 0;
    }
    if (board[pos] != '.') {
        print_out(0, "INVALID", "", expected_turn, "ERR_CELL_OCCUPIED", board);
        return 0;
    }

    char next_board[10];
    memcpy(next_board, board, 10);
    next_board[pos] = player[0];

    char w = detect_winner(next_board);
    if (w != 0) {
        char ws[2] = {w, 0};
        print_out(1, "WIN", ws, "", "GAME_WON", next_board);
        return 0;
    }
    if (x_count + o_count + 1 == 9) {
        print_out(1, "DRAW", "", "", "GAME_DRAW", next_board);
        return 0;
    }

    const char *nt = (player[0] == 'X') ? "O" : "X";
    print_out(1, "IN_PROGRESS", "", nt, "MOVE_ACCEPTED", next_board);
    return 0;
}
